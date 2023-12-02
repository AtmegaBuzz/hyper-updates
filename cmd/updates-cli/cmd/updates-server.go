package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use: "server",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

func trimNullChars(s string) string {

	t := strings.TrimRight(s, "\x00")
	u := strings.TrimLeft(t, "\x00")

	return u
}

func GetUpdateDataHandler(ctx context.Context) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		_, _, _, _, _, tcli, _ := handler.DefaultActor()

		t := r.URL.Query().Get("transactionid")
		transactionId, err := ids.FromString(t)

		if err != nil {

			fmt.Fprintln(w, "Invalid Id Passed")
			response := map[string]interface{}{
				"status": "failed",
			}

			w.Header().Set("Content-Type", "application/json")
			// w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		} else {

			_, ProjectTxID, UpdateExecutableHash, UpdateIPFSUrl, ForDeviceName, UpdateVersion, _, err := tcli.Update(ctx, transactionId, false)

			if err != nil {
				fmt.Fprintln(w, "Server Error")
			}

			response := map[string]interface{}{
				"ProjectTxID":          trimNullChars(string(ProjectTxID)),
				"UpdateExecutableHash": trimNullChars(string(UpdateExecutableHash)),
				"UpdateIPFSUrl":        trimNullChars(string(UpdateIPFSUrl)),
				"ForDeviceName":        trimNullChars(string(ForDeviceName)),
				"UpdateVersion":        UpdateVersion,
				"status":               "success",
			}
			fmt.Println("Project Tx Id: ", string(ProjectTxID), ", Exe Hash: ", string(UpdateExecutableHash), ", Ipfs URL: ", string(UpdateIPFSUrl), ", For Devide: ", string(ForDeviceName), ", Version: ", UpdateVersion)
			w.Header().Set("Content-Type", "application/json")

			// w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		}
	}

}

func GetUpdateHash(ctx context.Context) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		_, _, _, _, _, tcli, _ := handler.DefaultActor()

		t := r.URL.Query().Get("transactionid")
		hash := r.URL.Query().Get("hash")
		transactionId, err := ids.FromString(t)

		if err != nil {

			fmt.Fprintln(w, "Invalid Id Passed")
			response := map[string]interface{}{
				"status": "failed",
			}

			w.Header().Set("Content-Type", "application/json")
			// w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		} else {

			_, ProjectTxID, UpdateExecutableHash, UpdateIPFSUrl, ForDeviceName, UpdateVersion, _, err := tcli.Update(ctx, transactionId, false)

			if err != nil {
				fmt.Fprintln(w, "Server Error")
			}

			trueHash := trimNullChars(string(UpdateExecutableHash))
			response := ""
			if hash != trueHash {
				response = "INVALID"
			} else {
				response = "VALID"
			}

			fmt.Println("Project Tx Id: ", string(ProjectTxID), ", Exe Hash: ", string(UpdateExecutableHash), ", Ipfs URL: ", string(UpdateIPFSUrl), ", For Devide: ", string(ForDeviceName), ", Version: ", UpdateVersion)
			w.Header().Set("Content-Type", "application/json")

			// w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		}
	}

}

var startServer = &cobra.Command{
	Use: "start-server",
	RunE: func(*cobra.Command, []string) error {

		ctx := context.Background()

		// Register the handler function for the root ("/") route
		http.HandleFunc("/", GetUpdateDataHandler(ctx))
		http.HandleFunc("/check-hash", GetUpdateHash(ctx))

		// Start the HTTP server on port 8080
		fmt.Println("Server is listening on port 8080...")
		err_http := http.ListenAndServe(":8080", nil)

		if err_http != nil {
			return err_http
		}

		return err_http

	},
}
