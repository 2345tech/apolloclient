package apolloclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestPoll(t *testing.T) {
	client, err := NewClient("http://your-apollo.config-service.address", nil, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	param := GetConfigParam{
		AppID:     "demo_apollo_agent",
		Namespace: "application",
		Secret:    "a93ab23b59044e10b9bce61c4629e666",
	}

	data, err := client.GetConfigCache(&param)
	//data, err := client.GetConfig(&param)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(data.Configs)
}

func TestWatch(t *testing.T) {
	var body io.Reader
	var request *http.Request
	var err error
	var client *Client

	ctx, cancel := context.WithCancel(context.Background())
	request, err = http.NewRequestWithContext(ctx, http.MethodGet, "", body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	client, err = NewClient("http://your-apollo.config-service.address", nil, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	client.Request = request

	notificationParam := &GetNotificationsParam{
		AppID:         "demo_apollo_agent",
		Secret:        "a93ab23b59044e10b9bce61c4629e666",
		Notifications: make([]Notification, 0),
	}
	notificationParam.Notifications = []Notification{
		Notification{
			Namespace:      "application",
			NotificationID: 0,
		},
	}
	param := GetConfigParam{
		AppID:     "demo_apollo_agent",
		Namespace: "application",
		Secret:    "a93ab23b59044e10b9bce61c4629e666",
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("cancel request")
				return
			default:
				if update, notifications, err := client.GetNotifications(notificationParam); err != nil {
					fmt.Println("GetNotifications from Apollo Config Service error:" + err.Error())
				} else if update {
					notificationParam.Notifications[0].NotificationID = notifications[0].NotificationID
					if data, err := client.GetConfig(&param); err == nil {
						fmt.Println(data.Configs)
						param.ReleaseKey = data.ReleaseKey
					} else {
						fmt.Println("GetConfig from Apollo Config Service error:" + err.Error())
					}
				} else {
					fmt.Println("update = false")
				}
			}
		}
	}()

	time.Sleep(10 * time.Second)
	cancel()
	time.Sleep(1 * time.Second)
	fmt.Println("done")
}
