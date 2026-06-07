FROM node:18-alpine
WORKDIR /app
RUN echo "console.log('hello from docker build'); require('http').createServer((q,r)=>{r.end('ok')}).listen(3000)" > server.js
CMD ["node", "server.js"]
