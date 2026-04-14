### What is this?

HTTP file upload/download server written in Go. Upload any file, get a random URL, and share it with others. Perfect for quick file transfers between devices on your network.

### Features

- Upload files via HTTP POST
- Automatic random path generation for privacy
- Direct download links
- Smart handling: text files >10KB are forced to download instead of displaying in browser
- No size limit
- Request logging to console and file
- Configurable IP address and port (default 0.0.0.0:5555)
- IPv6 support

Linux:
```
wget -qO /tmp/up $(wget -qO- http://bit.ly/41R1Bxb | grep "browser_download_url" | grep -v ".exe" | cut -d '"' -f 4) && chmod +x /tmp/up && /tmp/up

or

curl -sL $(curl -sL http://bit.ly/41R1Bxb | grep "browser_download_url" | grep -v ".exe" | cut -d '"' -f 4) -o /tmp/up && chmod +x /tmp/up && /tmp/up
```

Windows powershell:
```
$u=(irm http://bit.ly/41R1Bxb).assets|?{$_.name -match "upserv.exe"};iwr -Uri $u.browser_download_url -OutFile "$env:TEMP\up.exe";Start-Process "$env:TEMP\up.exe"
```

## Usage:
Linux:
```
function up { [ -z "$UP_SERVER_IP" ] && read -p "Server IP: " UP_SERVER_IP; command -v curl >/dev/null || { apt update && apt install -y curl || yum install -y curl; }; [ -f "$1" ] && curl -F "file=@$1" "http://$UP_SERVER_IP:5555/upload" || echo "Upload failed"; }

up /path/to/file.txt
```

Windows powershell:
```
function up { param([string]$file); if (!(Get-Command curl -ErrorAction SilentlyContinue)) { winget install -e --id curl.curl }; & curl.exe -F "file=@$file" "http://<server-ip>:5555/upload" }

up "C:\path\to\file.txt"
```

Nginx sample location:
```
location / {
    proxy_pass http://127.0.0.1:5555;
    proxy_redirect http://127.0.0.1:5555 /;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Port $server_port;
}
```

Sample systemd unit file config:
```
[Unit]
Description=Run upserv at boot
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/upserv.sh
RemainAfterExit=true

[Install]
WantedBy=multi-user.target
```

upserv.sh:
```
#!/bin/bash
set -e

url=$(wget -qO- http://bit.ly/41R1Bxb | grep browser_download_url | grep -v .exe | cut -d '"' -f 4)
wget -qO /tmp/upserv "$url"
chmod +x /tmp/upserv
/tmp/upserv&
```

ShareX config (configure http proxy server first):
```
Name - <domain name>
Service Type - Image service / Text service / File service
Method - POST
Request URL - https://<domain name>/upload
Link - https://{regex:http:\/\/(?<url>.+)|url}
```

## Docker build

```sh
git clone https://github.com/attaattaatta/up.git
cd up/server/
docker run --rm -v "$PWD":/app -w /app golang:alpine go build -ldflags="-s -w" -o upserv upserv.go
```
