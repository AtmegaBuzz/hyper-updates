package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hyper-updates/actions"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
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
			http.Error(w, "Invalid TxId", http.StatusBadRequest)
			return

		} else {

			_, ProjectTxID, UpdateExecutableHash, UpdateIPFSUrl, ForDeviceName, UpdateVersion, _, err := tcli.Update(ctx, transactionId, false)

			if err != nil {
				fmt.Fprintln(w, "Server Error")
				http.Error(w, "Cannot query chain", http.StatusInternalServerError)
				return
			}

			trueHash := trimNullChars(string(UpdateExecutableHash))
			response := ""
			if hash != trueHash {
				http.Error(w, "Invalid String", http.StatusBadRequest)
				return
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

type ProjectInfo struct {
	ProjectName        string `json:"project_name"`
	ProjectDescription string `json:"project_description"`
	URL                string `json:"project_logo"`
}

func CreateRepositoryHandler(ctx context.Context) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()

		body, err := io.ReadAll(r.Body)

		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}

		var projectInfo ProjectInfo
		if err := json.Unmarshal(body, &projectInfo); err != nil {
			http.Error(w, "Error decoding JSON", http.StatusBadRequest)
			return
		}

		project := &actions.CreateProject{
			ProjectName:        []byte(projectInfo.ProjectName),
			ProjectDescription: []byte(projectInfo.ProjectDescription),
			Logo:               []byte(projectInfo.URL),
		}

		// Generate transaction
		_, id, err := sendAndWait(ctx, nil, project, cli, scli, tcli, factory, true)

		response := ""
		if err != nil {
			http.Error(w, "Error while creating Repository", http.StatusInternalServerError)
		}

		response = id.String()

		w.Header().Set("Content-Type", "application/json")

		// w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)

	}

}

func cleanupFile(file *os.File) {
	// Cleanup logic
	err := os.Remove(file.Name())
	if err != nil {
		fmt.Println("Error deleting file:", err)
		return
	}
	fmt.Println("File deleted successfully.")
}

func CreateUpdateHandler(ctx context.Context) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()

		err_mparse := r.ParseMultipartForm(10 << 20) // 10 MB limit for the entire request
		if err_mparse != nil {
			http.Error(w, "Unable to parse form", http.StatusBadRequest)
			return
		}

		// Extract form values
		projectID := r.FormValue("project_id")
		forDeviceName := r.FormValue("for_device_name")
		version, _ := strconv.ParseUint(r.FormValue("version"), 10, 8)

		// Get a reference to the uploaded file
		file, fileHeader, err := r.FormFile("executable_file")
		if err != nil {
			http.Error(w, "Unable to get file from request", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Create a new file on the server
		dst, err := os.Create(fileHeader.Filename)
		if err != nil {
			http.Error(w, "Unable to create file on server", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the new file
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Unable to copy file", http.StatusInternalServerError)
			return
		}

		executable_ipfs_url, err := DeployBin(
			fileHeader.Filename,
			"fc43a725fd778580045c",
			"37c52b3571d7df2c1326c1460a1b192c209a1fb212c6b1b96eb2626bb2076efe",
		)

		executable_hash, err := CalculateMD5(fileHeader.Filename)

		if err != nil {
			http.Error(w, "Cannot upload file to IPFS", http.StatusInternalServerError)
		}
		// Print received data
		fmt.Printf("Received data:\nProject ID: %s\nDevice Name: %s\nVersion: %s\n",
			projectID, forDeviceName, version)

		// File cleanup using defer
		defer cleanupFile(dst)

		update := &actions.CreateUpdate{
			ProjectTxID:          []byte(projectID),
			UpdateExecutableHash: []byte(executable_hash),
			UpdateIPFSUrl:        []byte(executable_ipfs_url),
			ForDeviceName:        []byte(forDeviceName),
			UpdateVersion:        uint8(version),
			SuccessCount:         0,
		}

		// Generate transaction
		_, id, err := sendAndWait(ctx, nil, update, cli, scli, tcli, factory, true)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("File uploaded successfully: " + id.String()))

	}

}

type PushUpdateInfo struct {
	UpdateTx string `json:"update-tx"`
	DeviceIp string `json:"device-ip"`
}

func pushFirmwareHash(hash, txid, filePath, deviceIp string) error {

	url := "http://" + deviceIp + "/ota/start?mode=fr&hash=" + hash + "&txid=" + txid
	fmt.Println(url)

	// Create the request
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}

	// Set the necessary headers
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0")
	request.Header.Set("Accept", "*/*")
	request.Header.Set("Accept-Language", "en-US,en;q=0.5")
	request.Header.Set("Accept-Encoding", "gzip, deflate")
	request.Header.Set("Referer", "http://192.168.0.6/update")
	request.Header.Set("Connection", "keep-alive")

	// Make the request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}
	defer response.Body.Close()

	// Print the response status and headers
	fmt.Println("Response Status:", response.Status)
	fmt.Println("Response Headers:", response.Header)

	// You can read and print the response body if needed
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return err
	}
	fmt.Println("Response Body:", string(body))

	return nil

}

func PushFirmwareUpdate(deviceIp string, filePath string, w http.ResponseWriter) error {

	url := "http://" + deviceIp + "/ota/upload"

	// Create a buffer to store the request body
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add the file to the request body
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Error Opening file", http.StatusInternalServerError)
		return err
	}
	defer file.Close()

	fileWriter, err := writer.CreateFormFile("file", "firmware.bin")
	if err != nil {
		fmt.Println("Error creating form file:", err)
		return err
	}

	_, err = io.Copy(fileWriter, file)
	if err != nil {
		fmt.Println("Error copying file content:", err)
		return err
	}

	// Close the multipart writer to finalize the request body
	writer.Close()

	// Create the request
	request, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}

	// Set the necessary headers
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "*/*")
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Referer", "http://"+deviceIp+"/update")
	request.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	request.Header.Set("Accept-Encoding", "gzip, deflate")

	// Make the request
	client := &http.Client{}

	// delete the downloaded firmware file
	deleteFile(filePath)

	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}
	defer response.Body.Close()

	// Read and print the response
	body2, err := io.ReadAll(response.Body)

	if err != nil {
		fmt.Println("Error reading response:", err)
		return err
	}

	fmt.Println("Response:", string(body2))

	return nil
}

func PushUpdate(ctx context.Context) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		filePath, err := os.Getwd()
		filePath += "/firmware.bin"

		_, _, _, _, _, tcli, err := handler.DefaultActor()

		body, err := io.ReadAll(r.Body)

		var pushUpdateInfo PushUpdateInfo
		if err := json.Unmarshal(body, &pushUpdateInfo); err != nil {
			http.Error(w, "Error decoding JSON", http.StatusBadRequest)
			return
		}

		transactionId, err := ids.FromString(pushUpdateInfo.UpdateTx)

		_, _, UpdateExecutableHash, UpdateIPFSUrl, _, _, _, err := tcli.Update(ctx, transactionId, false)

		if err != nil {
			fmt.Fprintln(w, "Server Error")
		}

		err_download := downloadIPFSFile(filePath, trimNullChars(string(UpdateIPFSUrl)))

		if err_download != nil {
			fmt.Println("Error Downloading file:", err)
			return
		}

		err = pushFirmwareHash(trimNullChars(string(UpdateExecutableHash)), transactionId.String(), filePath, pushUpdateInfo.DeviceIp)
		if err != nil {
			http.Error(w, "Cannot push hash to firmware: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = PushFirmwareUpdate(pushUpdateInfo.DeviceIp, filePath, w)
		if err != nil {
			http.Error(w, "Cannot push firmware: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Successfully Pushed updated"))
	}

}

func GetUpdate(ctx context.Context) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		_, _, _, _, _, tcli, _ := handler.DefaultActor()

		t := r.URL.Query().Get("transactionid")
		transactionId, _ := ids.FromString(t)

		_, ProjectTxID, UpdateExecutableHash, UpdateIPFSUrl, ForDeviceName, UpdateVersion, _, _ := tcli.Update(ctx, transactionId, false)

		uversion, _ := strconv.ParseUint(trimNullChars(string(UpdateVersion)), 10, 8)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Project Id: " + trimNullChars(string(ProjectTxID)) + "\n Hash: " + trimNullChars(string(UpdateExecutableHash)) + "\n IPFS URL: " + trimNullChars(string(UpdateIPFSUrl)) + "\n Device Name: " + trimNullChars(string(ForDeviceName)) + "\n VersionL " + string(uversion)))
	}

}

var startServer = &cobra.Command{
	Use: "start",
	RunE: func(*cobra.Command, []string) error {

		ctx := context.Background()

		http.HandleFunc("/", GetUpdateDataHandler(ctx))
		http.HandleFunc("/create-repository", CreateRepositoryHandler(ctx))
		http.HandleFunc("/create-update", CreateUpdateHandler(ctx))
		http.HandleFunc("/check-hash", GetUpdateHash(ctx))
		http.HandleFunc("/push-update", PushUpdate(ctx))
		http.HandleFunc("/get-update", GetUpdate(ctx))

		// Start the HTTP server on port 8080
		fmt.Println("Server is listening on port 8080...")
		err_http := http.ListenAndServe(":8080", nil)
		fmt.Println("Server Ended")

		if err_http != nil {
			return err_http
		}

		return err_http
	},
}
