FROM nginx:alpine

# Copy frontend files to nginx html directory
# Note: This should be built from the services/veladora directory with:
# docker build -f deploy/docker/frontend/Dockerfile -t veladora-frontend:latest .
COPY src/frontend/ /usr/share/nginx/html/

# Copy nginx configuration
COPY src/frontend/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

