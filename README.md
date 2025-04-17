# Tech Industry AI Usage Survey

A simple web-based survey application that collects responses about AI usage in the tech industry. The application is built with Go for the backend and uses HTML/JavaScript with Tailwind CSS for the frontend.

## Features

- Survey form with multiple-choice questions
- Real-time response visualization using Chart.js
- SQLite database for storing responses
- Rate limiting to prevent abuse
- Local storage to prevent multiple submissions from the same browser
- Support for "Others" option with custom text input

## Prerequisites

- Go 1.21 or later
- Modern web browser

## Project Structure

```
examensarbete-form/
├── form.go           # Main Go application
├── templates/        # HTML templates
│   └── index.html    # Main survey page
├── static/          # Static assets
└── database/        # SQLite database directory
```

## Quick Start

1. Clone the repository:
```bash
git clone <repository-url>
cd examensarbete-form
```

2. Build the application:
```bash
go build -o survey-app
```

3. Run the application:
```bash
./survey-app
```

The application will be available at `http://localhost:80`

## Deployment

### Simple Deployment (Development/Testing)

1. Build the application:
```bash
go build -o survey-app
```

2. Create necessary directories:
```bash
mkdir -p /opt/survey-app/database
mkdir -p /opt/survey-app/templates
mkdir -p /opt/survey-app/static
```

3. Copy the files:
```bash
cp survey-app /opt/survey-app/
cp -r templates static /opt/survey-app/
```

4. Run the application:
```bash
cd /opt/survey-app
./survey-app
```

### Production Deployment

For a production environment, it's recommended to use a process manager like systemd:

1. Create a systemd service file:
```bash
sudo nano /etc/systemd/system/survey-app.service
```

2. Add the following content:
```ini
[Unit]
Description=Survey App
After=network.target

[Service]
Type=simple
User=your-username
WorkingDirectory=/opt/survey-app
ExecStart=/opt/survey-app/survey-app
Restart=always

[Install]
WantedBy=multi-user.target
```

3. Enable and start the service:
```bash
sudo systemctl enable survey-app
sudo systemctl start survey-app
```

## Security Considerations

- The application includes basic rate limiting (10 requests per minute per IP)
- Form validation is implemented on both client and server side
- Local storage is used to prevent multiple submissions from the same browser
- SQLite database is used for simple data storage

## Monitoring

To check the application status:
```bash
sudo systemctl status survey-app
```

To view logs:
```bash
journalctl -u survey-app -f
```

## License

This project is licensed under the MIT License - see the LICENSE file for details. 