# go server + bash client uploader

## Запуск сервера:

Linux:
```
wget -qO /tmp/up $(wget -qO- http://bit.ly/41R1Bxb | grep "browser_download_url" | grep -v ".exe" | cut -d '"' -f 4) && chmod +x /tmp/up && /tmp/up
```
```
curl -sL $(curl -sL http://bit.ly/41R1Bxb | grep "browser_download_url" | grep -v ".exe" | cut -d '"' -f 4) -o /tmp/up && chmod +x /tmp/up && /tmp/up
```

Windows powershell:
```
$u=(irm http://bit.ly/41R1Bxb).assets|?{$_.name -match "upserv.exe"};iwr -Uri $u.browser_download_url -OutFile "$env:TEMP\up.exe";Start-Process "$env:TEMP\up.exe"
```

## Запуск клиента:
Linux:
```
up() { command -v curl >/dev/null || sudo apt update && sudo apt install -y curl; curl -F "file=@$1" http://<server-ip>:8080/upload; }
up file.txt
```

Windows powershell:
```
function up { param([string]$file); if (!(Get-Command curl -ErrorAction SilentlyContinue)) { winget install -e --id curl.curl }; & curl.exe -F "file=@$file" "http://<server-ip>:8080/upload" }
up "C:\path\to\file.txt"
```