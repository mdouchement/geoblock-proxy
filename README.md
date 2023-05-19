# geoblock-proxy

Simple Geoblocking Proxy that allows or blocks incoming connections for the specified countries or CIDR.
It works at `TCP` or `UDP` level.

This project relies IP2Location LITE data available from [`lite.ip2location.com`](https://lite.ip2location.com/database/ip-country) database
- Databases: [https://download.ip2location.com/lite](https://download.ip2location.com/lite/)
- `go run .tools/ip2location-download/main.go https://download.ip2location.com/lite/IP2LOCATION-LITE-DB1.BIN.ZIP IP2LOCATION-LITE-DB1.BIN`


## License

**MIT**


## Contributing

All PRs are welcome.

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Add some feature')
5. Push to the branch (git push origin my-new-feature)
6. Create new Pull Request