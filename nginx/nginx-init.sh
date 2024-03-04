echo "
server {
  listen 80;
  server_name $DOMAIN;
  location / {
    proxy_pass http://localhost:3200;
  }
}
" > /etc/nginx/conf.d/default.conf


cat /etc/nginx/conf.d/default.conf

certbot --nginx -m jhartway99@gmail.com --agree-tos --no-eff-email -d $DOMAIN --redirect --non-interactive
