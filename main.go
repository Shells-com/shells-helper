package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/TrisTech/goupd"
	"github.com/godbus/dbus/v5"
)

func main() {
	goupd.AutoUpdate(false)
	var lastSummary, lastBody string

	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
		os.Exit(1)
	}
	defer conn.Close()

	var rules = []string{
		"type='signal',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
		"type='method_call',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
		"type='method_return',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
		"type='error',member='Notify',path='/org/freedesktop/Notifications',interface='org.freedesktop.Notifications'",
	}
	var flag uint = 0

	call := conn.BusObject().Call("org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, flag)
	if call.Err != nil {
		fmt.Fprintln(os.Stderr, "Failed to become monitor:", call.Err)
		os.Exit(1)
	}

	c := make(chan *dbus.Message, 10)
	conn.Eavesdrop(c)
	log.Printf("This program monitors notifications on dbus and send them to Shells system for forwarding to browser/etc")
	for v := range c {
		//log.Printf("got notification: %T %+v", v, v)

		if v.Type == dbus.TypeMethodCall {
			summary := v.Body[3].(string)
			body := v.Body[4].(string)

			if summary == lastSummary && body == lastBody {
				// skip duplicate
				continue
			}

			lastSummary = summary
			lastBody = body

			//log.Printf("Summary=%q Body=%q", summary, body)

			sendNotify(map[string]interface{}{"title": summary, "body": body})
		}

		// Body: 0:AppName 1:ReplacesID 2:AppIcon 3:Summary 4:Body 5:[]Actions 6:Hints[string]Variant 7:ExpireTimeout
	}
}

func sendNotify(req map[string]interface{}) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Post("http://169.254.169.254/private/shells/notify", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	ioutil.ReadAll(resp.Body)
	//body, err := ioutil.ReadAll(resp.Body)
	//log.Printf("res=%s err=%v", body, err)
	return nil
}
