// Rest API Implementations

package main

import (
	"encoding/json"
	"fmt"
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

		switch {
		case err == nil:
			return fmt.Sprintf("Device '%s' with IP '%s' is awake. Packet arp ping time '%s'", host, ips[idx], time), nil
		case err == arping.ErrTimeout:
			status = fmt.Sprintf("Device '%s' with IP '%s' is offline", host, ips[idx])
		case err.Error() == "interrupted system call":
			status = fmt.Sprintf("Device '%s' with IP '%s' is offline", host, ips[idx])
		default:
			status = fmt.Sprintf("Error: '%s' while sending arping to device '%s'", err.Error(), host)
			// Get root cause in readable format
			errMsg = fmt.Errorf(err.Error())
		}
	}
	return status, errMsg
}

// restWakeUpWithDeviceName - REST Handler for Processing URLS /virtualdirectory/apipath/<deviceName>
func wakeUpWithDeviceName(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	deviceName := vars["deviceName"]

	var errMsg string
	var o HTTPResponseObject
	o.Success = false

	// Ensure deviceName is not empty
	if deviceName == "" {
		// Devicename is empty
		o.Message = "Error: Empty device names are not allowed."
		o.ErrorObject = nil
		w.WriteHeader(http.StatusBadRequest)
	} else {
		// Get Device from List
		for _, c := range appData.Devices {
			if c.Name == deviceName {
				// We found the Devicename
				if err := SendMagicPacket(c.Mac, c.BroadcastIP, ""); err != nil {
					// We got an internal Error on SendMagicPacket
					w.WriteHeader(http.StatusInternalServerError)
					o.Success = false
					o.Message = "Error: internal error while sending the Magic Packet."
					o.ErrorObject = err
				} else {
					// Horray we send the WOL Packet succesfully
					o.Success = true
					o.Message = fmt.Sprintf("Sent magic packet to device '%s' with MAC '%s' on Broadcast IP '%s'.", c.Name, c.Mac, c.BroadcastIP)
					o.ErrorObject = nil
				}
			}
		}
	}

	switch {
	case !o.Success && o.ErrorObject == nil:
		// We could not find the Devicename
		w.WriteHeader(http.StatusNotFound)
		o.Message = fmt.Sprintf("Error: Device name '%s' could not be found.", deviceName)
	case o.Success:
		// Sending MagicPacket was success. Now let's arping the host
		o.Message, o.ErrorObject = arpingHost(deviceName)
		if o.ErrorObject != nil {
			o.Success = false
			errMsg = o.ErrorObject.Error()
		}
	case o.ErrorObject != nil:
		errMsg = o.ErrorObject.Error()
	}
	json.NewEncoder(w).Encode(MsgSerialization{
		Message: o.Message,
		Error:   errMsg,
		Success: o.Success})
}
