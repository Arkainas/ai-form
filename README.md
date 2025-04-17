# Tech Industry AI Usage Survey

A simple web application that collects responses about AI usage in the tech industry. The application ensures that each IP address can only submit the form once.

## Features
- Single submission per IP address
- SQLite database for response storage
- Simple HTML/JavaScript frontend
- Go backend server

## Setup
1. Make sure you have Go installed on your system
2. Install the required dependencies:
```bash
go mod init tech-survey
go mod tidy
```
3. Run the server:
```bash
go run .
```
4. Visit `http://localhost:8080` in your browser

## Project Structure
- `form.go` - Main Go server file
- `static/` - Frontend static files
- `database/` - SQLite database
- `templates/` - HTML templates 