package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Arg map for templates
type M map[string]interface{}

type StatusResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func checkServerAuth(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || len(token) != 40 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	log.Debugf("Authed as: %s", token)
	if !store.TokenValid(token) {
		log.Warnf("Received invalid server token from %s", c.ClientIP())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	c.Next()
}

func initHTTP() {
	log.Infof("Starting gbans HTTP service")
	go func() {
		if err := router.Run(config.HTTP.Addr()); err != nil {
			log.Errorf("Error shutting down service: %v", err)
		}
	}()
	<-ctx.Done()
}

func routeRaw(name string) string {
	routePath, ok := routes[Route(name)]
	if !ok {
		return "/xxx"
	}
	return routePath
}

// route will return a route for the simple name provided. If the route has parameters, the function
// will ensure that they are supplied.
func route(name string, args ...interface{}) string {
	const sep = ":"
	routePath := routeRaw(name)
	if !strings.Contains(routePath, sep) {
		return routePath
	}
	cnt := strings.Count(routePath, sep)
	if len(args) != cnt {
		log.Errorf("Route args count mismatch. Have: %d Want: %d", len(args), cnt)
		return routePath
	}
	varIdx := 0
	p := strings.Split(routePath, "/")
	p = p[1:]
	for i, part := range p {
		if strings.HasPrefix(part, sep) {
			p[i] = fmt.Sprintf("%s", args[varIdx])
			varIdx++
			if varIdx == len(args) {
				break
			}
		}
	}
	return "/" + strings.Join(p, "/")
}

func authMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		s := sessions.Default(c)
		guest := model.NewPlayer()
		var p model.Player
		var err error
		v := s.Get("person_id")
		if v != nil {
			pId, ok := v.(int)
			if ok {
				p, err = store.GetPersonBySteamID(steamid.SID64(pId))
				if err != nil {
					log.Errorf("Failed to load persons session user: %v", err)
					p = guest
				}
			} else {
				// Delete the bad value
				s.Delete("person_id")
				if err := s.Save(); err != nil {
					log.Errorf("Failed to save session")
				}
			}
		} else {
			p = guest
		}
		c.Set("person", p)
		c.Next()
	}
}

func currentPerson(c *gin.Context) model.Player {
	p, found := c.Get("person")
	if !found {
		return model.NewPlayer()
	}
	person, ok := p.(model.Player)
	if !ok {
		log.Warnf("Count not cast store.Player from session")
		return model.NewPlayer()
	}
	return person
}

func defaultArgs(c *gin.Context) M {
	args := M{}
	args["site_name"] = config.HTTP.SiteName
	args["person"] = currentPerson(c)
	return args
}

func newTmpl(files ...string) *template.Template {
	var tFuncMap = template.FuncMap{
		"icon": func(class string) template.HTML {
			return template.HTML(fmt.Sprintf(`<i class="%s"></i>`, class))
		},
		"currentYear": func() template.HTML {
			return template.HTML(fmt.Sprintf("%d", time.Now().UTC().Year()))
		},
		"datetime": func(t time.Time) template.HTML {
			return template.HTML(t.Format(time.RFC822))
		},
		"fmtFloat": func(f float64, size int) template.HTML {
			ft := fmt.Sprintf("%%.%df", size)
			return template.HTML(fmt.Sprintf(ft, f))
		},
		"route": func(name string) template.HTML {
			return template.HTML(route(name))
		},
	}
	tmpl, err := template.New("layout").Funcs(tFuncMap).ParseFiles(files...)
	if err != nil {
		log.Panicf("Failed to load template: %v", err)
	}
	return tmpl
}

func initTemplates() {
	var templateFiles []string
	root := "templates"
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".gohtml") {
			if !strings.Contains(path, "_") && !strings.Contains(path, "/partials") {
				templateFiles = append(templateFiles, info.Name())
			}
		}
		return nil
	}); err != nil {
		log.Fatalf("Failed to read templates: %v", err)
	}
	var newPagesSet = func(path string) []string {
		return []string{
			fmt.Sprintf("templates/%s.gohtml", path),
			//"templates/partials/page_header.gohtml",
			"templates/_layout.gohtml",
		}
	}
	for _, p := range templateFiles {
		pageN := strings.ReplaceAll(p, ".gohtml", "")
		templates[pageN] = newTmpl(newPagesSet(pageN)...)
	}
}

func render(c *gin.Context, t string, args M) {
	var buf bytes.Buffer
	tmpl := templates[t]
	if err := tmpl.ExecuteTemplate(&buf, "layout", args); err != nil {
		log.Errorf("Failed to execute template: %v", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Data(200, gin.MIMEHTML, buf.Bytes())
}

func onIndex() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "home", defaultArgs(c))
	}
}

func onPostAuth() gin.HandlerFunc {
	type authReq struct {
		ServerName string `json:"server_name"`
		Key        string `json:"key"`
	}
	type authResp struct {
		Status bool   `json:"status"`
		Token  string `json:"token"`
	}
	return func(c *gin.Context) {
		var req authReq
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("Failed to decode auth request: %v", err)
			c.JSON(500, authResp{Status: false})
			return
		}
		srv, err := store.GetServerByName(req.ServerName)
		if err != nil {
			c.JSON(http.StatusNotFound, authResp{Status: false})
			return
		}
		srv.Token = golib.RandomString(40)
		srv.TokenCreatedOn = time.Now().Unix()
		if err := store.SaveServer(&srv); err != nil {
			log.Errorf("Failed to updated server token: %v", err)
			c.JSON(500, authResp{Status: false})
			return
		}
		c.JSON(200, authResp{
			Status: true,
			Token:  srv.Token,
		})
	}
}

func onPostBan() gin.HandlerFunc {
	type req struct {
		SteamID    string        `json:"steam_id"`
		AuthorID   string        `json:"author_id"`
		Duration   string        `json:"duration"`
		IP         string        `json:"ip"`
		BanType    model.BanType `json:"ban_type"`
		Reason     model.Reason  `json:"reason"`
		ReasonText string        `json:"reason_text"`
	}

	return func(c *gin.Context) {
		var r req
		if err := c.BindJSON(&r); err != nil {
			c.JSON(http.StatusBadRequest, StatusResponse{
				Success: false,
				Message: "Failed to perform ban",
			})
			return
		}
		duration, err := time.ParseDuration(r.Duration)
		if err != nil {
			c.JSON(http.StatusNotAcceptable, StatusResponse{
				Success: false,
				Message: `Invalid duration. Examples: "300m", "1.5h" or "2h45m". 
Valid time units are "s", "m", "h".`,
			})
		}
		ip := net.ParseIP(r.IP)
		if err := Ban(c, r.SteamID, r.AuthorID, duration, ip, r.BanType, r.Reason, r.ReasonText, model.Web); err != nil {
			c.JSON(http.StatusNotAcceptable, StatusResponse{
				Success: false,
				Message: "Failed to perform ban",
			})
		}
	}
}

func onPostCheck() gin.HandlerFunc {
	type checkRequest struct {
		ClientID int    `json:"client_id"`
		SteamID  string `json:"steam_id"`
		IP       string `json:"ip"`
	}
	type checkResponse struct {
		ClientID int           `json:"client_id"`
		SteamID  string        `json:"steam_id"`
		BanType  model.BanType `json:"ban_type"`
		Msg      string        `json:"msg"`
	}
	return func(c *gin.Context) {
		var req checkRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(500, checkResponse{
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			return
		}
		resp := checkResponse{
			ClientID: req.ClientID,
			SteamID:  req.SteamID,
			BanType:  model.Unknown,
			Msg:      "",
		}
		// Check IP first
		banNet, err := store.GetBanNet(req.IP)
		if err == nil {
			resp.BanType = model.Banned
			resp.Msg = banNet.Reason
			c.JSON(200, resp)
			return
		}
		// Check SteamID
		steamID, err := steamid.ResolveSID64(context.Background(), req.SteamID)
		if err != nil || !steamID.Valid() {
			resp.Msg = "Invalid steam id"
			c.JSON(500, resp)
		}
		ban, err := store.GetBan(steamID)
		if err != nil {
			if store.DBErr(err) == store.ErrNoResult {
				resp.BanType = model.OK
				c.JSON(200, resp)
				return
			}
			resp.Msg = "Error determining state"
			c.JSON(500, resp)
			return
		}
		resp.BanType = ban.BanType
		resp.Msg = ban.ReasonText
		c.JSON(200, resp)
	}
}

func onGetBan() gin.HandlerFunc {
	type banStateRequest struct {
		SteamID string `json:"steam_id"`
	}
	type banStateResponse struct {
		SteamID string        `json:"steam_id"`
		BanType model.BanType `json:"ban_type"`
		Msg     string        `json:"msg"`
	}
	return func(c *gin.Context) {
		var req banStateRequest

		if err := c.BindJSON(&req); err != nil {
			c.JSON(500, banStateResponse{
				SteamID: "",
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			return
		}
		c.JSON(200, gin.H{"status": model.OK})
	}
}
