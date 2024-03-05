#!/bin/bash -xe
apt-get update
apt-get install -y docker.io git nginx
systemctl enable docker
systemctl start docker

echo "${private_key}" > /etc/ssl/private/certbot-nginx.key
echo "${cert}" > /etc/ssl/certs/certbot-nginx.crt

cat <<'EOF' > /etc/nginx/sites-available/default 

  server {
    listen 80;
    server_name ${domain};
    return 301 https://$server_name$request_uri;
  } 
  server {
    listen 443 ssl;
    server_name ${domain};
    ssl_certificate /etc/ssl/certs/certbot-nginx.crt;
    ssl_certificate_key /etc/ssl/private/certbot-nginx.key;
  
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers "ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-DSS-AES128-GCM-SHA256:kEDH+AESGCM:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA:ECDHE-ECDSA-AES256-SHA:DHE-RSA-AES256-SHA256:DHE-RSA-AES256-SHA:DHE-DSS-AES256-SHA256:AES256-GCM-SHA384:AES128-GCM-SHA256:AES256-SHA256:AES128-SHA256:AES256-SHA:AES128-SHA:DHE-RSA-AES256-SHA:DHE-DSS-AES256-SHA:DHE-RSA-AES128-SHA:DHE-DSS-AES128-SHA:HIGH:!aNULL:!eNULL:!EXPORT:!DES:!MD5:!PSK:!RC4";

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    location / {
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_pass http://localhost:3200/;
    }
  }

EOF

systemctl restart nginx

docker run -d --network host jhartway99/htmx-flappybird
git clone https://github.com/DeaSTL/flappybird-htmx
