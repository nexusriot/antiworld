# Antiworld3
Tool for downloading books from _https://fantasy-worlds.org_ website.

>**Warning:**
>Made for educational purposes only under MIT license.
>If you like the book please buy it.


Building (linux, static):
```
go build -ldflags "-linkmode external -extldflags -static"
```

Building (windows):

```
GOOS=windows GOARCH=amd64 go build -o antiworld3.exe
```

Configuration file example, use encryptor to get encrypted password.
```
{
    "download_folder": "_library/",
    "base_url": "https://fantasy-worlds.org",
    "proxy" : {
                "address": "10.0.0.5:1080",
                "username": "proxy_username",
                "password": "Pta76Ta71a++24-3P77vPI="
    }   
}
```
Proxy section is optional (currently supported only SOCKS5), so if you don't want to use SOCKS5 proxy, just ignore it.