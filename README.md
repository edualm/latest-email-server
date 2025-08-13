# Latest Email Server

A simple Go web server that connects to an IMAP server, retrieves the latest email from your inbox, and displays it in your browser with proper HTML rendering.

## Features

- Connects to any IMAP server with TLS support;
- Retrieves the latest email from your inbox;
- Renders HTML emails properly in your browser;
- Decodes quoted-printable encoding;
- Configurable via JSON config file;
- Lightweight and fast.

## Setup

1. Clone this repository;
2. Install Go dependencies:
   ```bash
   go mod tidy
   ```

3. Create a `config.json` file with your email settings:
   ```json
   {
     "imap_server": "imap.example.com:993",
     "email": "your-email@example.com", 
     "password": "your-password",
     "listen_port": "8080"
   }
   ```

4. Build and run the server:
   ```bash
   go build
   ./latest-email-server
   ```

5. (Optional) Edit the systemd service file and install it;

6. Open your browser and go to `http://localhost:8080` to see your latest email.

## Configuration

Edit `config.json` to configure:

- `imap_server`: IMAP server address with port (e.g., "imap.gmail.com:993");
- `email`: Your email address;
- `password`: Your email password or app-specific password;
- `listen_port`: Port for the web server to listen on.

## Security Notes

- Use app-specific passwords when available (Gmail, Outlook, etc.);
- The server has no authentication is intended for local/private use.

## Dependencies

- [go-imap/v2](https://github.com/emersion/go-imap) - IMAP client library.

## License

MIT License