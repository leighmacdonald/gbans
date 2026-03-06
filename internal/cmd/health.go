package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/spf13/cobra"
)

var ErrHealth = errors.New("healthcheck failed")

func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check health",
		Long:  `Check the current health of the running app. Meant for use with dockers healthcheck.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			static, errStatic := config.ReadStaticConfig()
			if errStatic != nil {
				return errors.Join(errStatic, ErrHealth)
			}

			appURL := fmt.Sprintf("http://%s:%d", static.HTTPHost, static.HTTPPort)
			req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, appURL, nil)
			if errReq != nil {
				return errors.Join(errReq, ErrHealth)
			}

			client := http.Client{}
			resp, errResp := client.Do(req)
			if errResp != nil {
				return errors.Join(errResp, ErrHealth)
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("%w: Invalid response code: %d", ErrHealth, resp.StatusCode)
			}

			return nil
		},
	}
}
