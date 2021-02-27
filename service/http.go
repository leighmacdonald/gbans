package service

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	log "github.com/sirupsen/logrus"
	"html/template"
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
	if !tokenValid(token) {
		log.Warnf("Received invalid server token from %s", c.ClientIP())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	c.Next()
}

func initHTTP() {
	log.Infof("Starting HTTP service")
	go func() {
		httpServer = &http.Server{
			Addr:           config.HTTP.Addr(),
			Handler:        router,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		if config.HTTP.TLS {
			tlsVar := &tls.Config{
				// Causes servers to use Go's default ciphersuite preferences,
				// which are tuned to avoid attacks. Does nothing on clients.
				PreferServerCipherSuites: true,
				// Only use curves which have assembly implementations
				CurvePreferences: []tls.CurveID{
					tls.CurveP256,
					tls.X25519, // Go 1.8 only
				},
				MinVersion: tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				},
			}
			httpServer.TLSConfig = tlsVar
		}
		if err := httpServer.ListenAndServe(); err != nil {
			log.Errorf("Error shutting down service: %v", err)
		}
	}()
	<-ctx.Done()
}

func routeRaw(name string) string {
	routePath, ok := routes[routeKey(name)]
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
		log.Errorf("routeKey args count mismatch. Have: %d Want: %d", len(args), cnt)
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

func currentPerson(c *gin.Context) model.Person {
	p, found := c.Get("person")
	if !found {
		return model.NewPerson()
	}
	person, ok := p.(model.Person)
	if !ok {
		log.Warnf("Total not cast store.Person from session")
		return model.NewPerson()
	}
	return person
}

type TmplArgs struct {
	Person   model.Person
	SiteName string
	V        M
	Flashes  []Flash
}

func getFlashes(c *gin.Context) []Flash {
	var flashes []Flash
	session := sessions.Default(c)
	for _, flash := range session.Flashes() {
		f, ok := flash.(Flash)
		if !ok {
			log.Errorf("failed to cast flash??")
		}
		flashes = append(flashes, f)
	}
	if err := session.Save(); err != nil {
		log.Errorf("Failed to save session after flashes: %v", err)
	}
	return flashes
}

func defaultArgs(c *gin.Context) TmplArgs {
	args := TmplArgs{}
	args.SiteName = config.General.SiteName
	args.Person = currentPerson(c)
	args.Flashes = getFlashes(c)
	args.V = M{}
	return args
}

func newTmpl(files ...string) *template.Template {
	var tFuncMap = template.FuncMap{
		"icon": func(class string) template.HTML {
			return template.HTML(fmt.Sprintf(`<i class="%s"></i>`, class))
		},
		"currentYear": func() template.HTML {
			return template.HTML(fmt.Sprintf("%d", config.Now().Year()))
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
			if !strings.HasPrefix(path, "_") && !strings.Contains(path, "/partials") {
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
	var tpls []string
	for k := range templates {
		if !strings.Contains(k, "%s") && !strings.HasPrefix(k, "_") {
			tpls = append(tpls, k)
		}
	}
	log.Debugf("Loaded templates: %s", strings.Join(tpls, ", "))
}

func render(c *gin.Context, t string, args TmplArgs) {
	var buf bytes.Buffer
	tmpl := templates[t]
	if err := tmpl.ExecuteTemplate(&buf, "layout", args); err != nil {
		log.Errorf("Failed to execute template: %v", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Data(200, gin.MIMEHTML, buf.Bytes())
}
