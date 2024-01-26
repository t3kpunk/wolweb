// Rest API Implementations

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/j-keck/arping"
)

type MsgSerialization struct {
	Message string `json:"message"`
	Error   string `json:"error"`
	Success bool   `json:"success"`
}

func arpingHost(host string) (string, error) {

	var status string
	var errMsg error

	ips, err := net.LookupIP(host)
	if err != nil {
		status = fmt.Sprintf("Error: device '%s' could not be resolved", host)
		errMsg = err
	}

	for idx := range ips {
		_, time, err := arping.Ping(ips[idx])
		if err == arping.ErrTimeout {
			status = fmt.Sprintf("Device '%s' with IP '%s' is offline", host, ips[idx])
		} else if err != nil {
			status = fmt.Sprintf("Error: '%s' while sending arping to device '%s'", err.Error(), host)
			// Get cause in text
			errMsg = fmt.Errorf(err.Error())
		} else {
			return fmt.Sprintf("Device '%s' with IP '%s' is awake. Packet took '%s'", host, ips[idx], time), nil
		}
	}
	return status, errMsg
}

// restWakeUpWithDeviceName - REST Handler for Processing URLS /virtualdirectory/apipath/<deviceName>
func wakeUpWithDeviceName(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	deviceName := vars["deviceName"]

	var errStr string
	var result HTTPResponseObject
	result.Success = false

	// Ensure deviceName is not empty
	if deviceName == "" {
		// Devicename is empty
		result.Message = "Error: Empty device names are not allowed."
		result.ErrorObject = nil
		w.WriteHeader(http.StatusBadRequest)
	} else {

		// Get Device from List
		for _, c := range appData.Devices {
			if c.Name == deviceName {

				// We found the Devicename
				if err := SendMagicPacket(c.Mac, c.BroadcastIP, ""); err != nil {
					// We got an internal Error on SendMagicPacket
					w.WriteHeader(http.StatusInternalServerError)
					result.Success = false
					result.Message = "Error: internal error while sending the Magic Packet."
					result.ErrorObject = err
				} else {
					// Horray we send the WOL Packet succesfully
					result.Success = true
					result.Message = fmt.Sprintf("Sent magic packet to device '%s' with MAC '%s' on Broadcast IP '%s'.", c.Name, c.Mac, c.BroadcastIP)
					result.ErrorObject = nil
				}
			}
		}

		if !result.Success && result.ErrorObject == nil {
			// We could not find the Devicename
			w.WriteHeader(http.StatusNotFound)
			result.Message = fmt.Sprintf("Error: Device name '%s' could not be found.", deviceName)
		}

		if result.Success {
			log.Printf(result.Message)
			status, err := arpingHost(deviceName)
			result.Message = status
			if err != nil {
				result.ErrorObject = err
				result.Success = false
			}
		}
	}

	if result.ErrorObject != nil {
		errStr = result.ErrorObject.Error()
	}
	json.NewEncoder(w).Encode(MsgSerialization{
		Message: result.Message,
		Error:   errStr,
		Success: result.Success})
}
