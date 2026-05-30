package cmd

// Template represents a project template.
type Template struct {
	Name        string
	Description string
	Files       map[string]string
}

var templates = map[string]Template{
	"react-express-postgres": {
		Name:        "react-express-postgres",
		Description: "React frontend + Express API + PostgreSQL database",
		Files: map[string]string{
			"devbox.yml": `name: react-express-postgres
version: "1.0"
services:
  web:
    image: node:20-alpine
    command: npm run dev
    port: "5173:5173"
    working_dir: /app/web
    volumes:
      - ./web:/app/web
    depends_on: [api]
  api:
    image: node:20-alpine
    command: npm run dev
    port: "3001:3000"
    working_dir: /app/api
    volumes:
      - ./api:/app/api
    depends_on: [db]
  db:
    image: postgres:16-alpine
    port: "5432:5432"
    env:
      POSTGRES_DB: app
      POSTGRES_USER: app
      POSTGRES_PASSWORD: devbox_local_dev
volumes:
  pgdata:
`,
			"docker-compose.override.yml": `version: "3.8"
services:
  db:
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
`,
			"web/package.json": `{
  "name": "web",
  "private": true,
  "scripts": {
    "dev": "vite",
    "build": "vite build"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "vite": "^5.0.0",
    "@vitejs/plugin-react": "^4.0.0"
  }
}`,
			"api/package.json": `{
  "name": "api",
  "private": true,
  "scripts": {
    "dev": "node --watch server.js"
  },
  "dependencies": {
    "express": "^4.18.0",
    "pg": "^8.11.0"
  }
}`,
		},
	},
	"go-api": {
		Name:        "go-api",
		Description: "Go HTTP API server",
		Files: map[string]string{
			"devbox.yml": `name: go-api
version: "1.0"
services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    command: go run main.go
    port: "8080:8080"
    volumes:
      - .:/app
    working_dir: /app
`,
			"Dockerfile": `FROM golang:1.22-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/server .
CMD ["/app/server"]
`,
			"main.go": `package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from DevBoxOS Go API!")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	fmt.Printf("Server listening on :%s\n", port)
	http.ListenAndServe(":"+port, nil)
}
`,
			"go.mod": `module github.com/example/go-api

go 1.22
`,
		},
	},
	"python-django": {
		Name:        "python-django",
		Description: "Python Django web application",
		Files: map[string]string{
			"devbox.yml": `name: python-django
version: "1.0"
services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
    command: python manage.py runserver 0.0.0.0:8000
    port: "8000:8000"
    volumes:
      - .:/app
    working_dir: /app
    depends_on: [db]
  db:
    image: postgres:16-alpine
    port: "5432:5432"
    env:
      POSTGRES_DB: django
      POSTGRES_USER: django
      POSTGRES_PASSWORD: devbox_local_dev
`,
			"Dockerfile": `FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
CMD ["python", "manage.py", "runserver", "0.0.0.0:8000"]
`,
			"requirements.txt": `Django>=5.0,<6.0
psycopg2-binary>=2.9,<3.0
`,
			"manage.py": `#!/usr/bin/env python
"""Django's command-line utility for administrative tasks."""
import os
import sys

def main():
    os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'app.settings')
    from django.core.management import execute_from_command_line
    execute_from_command_line(sys.argv)

if __name__ == '__main__':
    main()
`,
		},
	},
	"node-express": {
		Name:        "node-express",
		Description: "Node.js Express API server",
		Files: map[string]string{
			"devbox.yml": `name: node-express
version: "1.0"
services:
  api:
    image: node:20-alpine
    command: npm run dev
    port: "3000:3000"
    volumes:
      - .:/app
    working_dir: /app
`,
			"package.json": `{
  "name": "node-express-api",
  "private": true,
  "scripts": {
    "dev": "node --watch server.js"
  },
  "dependencies": {
    "express": "^4.18.0"
  }
}`,
			"server.js": `const express = require('express');
const app = express();
const port = process.env.PORT || 3000;

app.get('/', (req, res) => {
  res.json({ message: 'Hello from DevBoxOS!' });
});

app.get('/health', (req, res) => {
  res.status(200).send('ok');
});

app.listen(port, () => {
  console.log(Server listening on port ${port});
});
`,
		},
	},
	"rust-axum": {
		Name:        "rust-axum",
		Description: "Rust Axum web server",
		Files: map[string]string{
			"devbox.yml": `name: rust-axum
version: "1.0"
services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    command: cargo run
    port: "8080:8080"
    volumes:
      - .:/app
    working_dir: /app
`,
			"Dockerfile": `FROM rust:1.75-slim-bookworm
WORKDIR /app
COPY Cargo.toml Cargo.lock* ./
RUN cargo fetch
COPY . .
CMD ["cargo", "run"]
`,
			"Cargo.toml": `[package]
name = "rust-axum-api"
version = "0.1.0"
edition = "2021"

[dependencies]
axum = "0.7"
tokio = { version = "1", features = ["full"] }
serde = { version = "1", features = ["derive"] }
`,
			"src/main.rs": `use axum::{routing::get, Router};

#[tokio::main]
async fn main() {
    let app = Router::new()
        .route("/", get(|| async { "Hello from DevBoxOS Rust API!" }))
        .route("/health", get(|| async { "ok" }));

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
`,
		},
	},
}
