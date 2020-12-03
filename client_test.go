package apolloclient

import (
	"fmt"
	"os"
	"testing"
)

func TestGetConfigCache(t *testing.T) {
	client, err := NewConfigService(nil, "http://172.17.212.3:8080")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	param := GetConfigParam{
		AppID:     "demo_apollo_agent",
		Namespace: "application",
	}

	configData, err := client.GetConfigCache(&param)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(configData.Configs)
}
