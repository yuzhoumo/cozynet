# CozyNet Search Frontend

A static search frontend for the CozyNet search engine.

## Features

- Clean, responsive search interface
- Real-time search with pagination
- Domain and word count filtering
- Search statistics display
- Mobile-friendly design

## Usage

1. Start the Greenhouse API server:
   ```bash
   cd ../greenhouse
   python main.py
   ```

2. Serve the frontend files (any HTTP server works):
   ```bash
   # Using Python
   python -m http.server 8080
   
   # Using Node.js
   npx serve .
   
   # Using PHP
   php -S localhost:8080
   ```

3. Open http://localhost:8080 in your browser

## API Configuration

The frontend expects the API to be running on `http://localhost:8000`. To change this, edit the `apiBaseUrl` in `app.js`.

## Files

- `index.html` - Main page structure
- `app.js` - Search functionality and API communication
- `style.css` - Responsive styling
- `README.md` - This documentation