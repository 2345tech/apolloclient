## apolloClient

Apollo Go Client

[![GitHub Release](https://img.shields.io/github/release/2345tech/apolloclient.svg)](https://github.com/2345tech/apolloclient/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)


## feature
```go
GetConfig(param *GetConfigParam) (ConfigData, error)
GetConfigCache(param *GetConfigParam) (ConfigData, error)
GetNotifications(param *GetNotificationsParam) (bool, []Notification, error)

// support set access_key for configService
signature(secret, uri string, timestamp time.Time) (string, error)
```

## example

https://github.com/2345tech/apollo-agent