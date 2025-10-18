# GoodPack Server

A Go-based REST API server for inventory management with MongoDB integration.

## 🚀 Features

- **Product Management**: CRUD operations for products
- **Inventory Tracking**: Stock management and reporting
- **QR Code Generation**: Generate QR codes for products
- **MongoDB Integration**: Persistent data storage
- **CORS Support**: Cross-origin resource sharing
- **Environment Configuration**: Configurable via environment variables

## 📋 Prerequisites

- Go 1.21 or higher
- MongoDB 4.4 or higher

## 🛠️ Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd goodpack-server
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Start MongoDB**
   ```bash
   # Using Docker
   docker run -d -p 27017:27017 --name mongodb mongo:latest
   
   # Or install MongoDB locally
   # Follow MongoDB installation guide for your OS
   ```

4. **Create environment file**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

5. **Run the server**
   ```bash
   go run main.go
   ```

## 🔧 Configuration

Create a `.env` file in the root directory:

```env
# Server Configuration
PORT=8080
ENVIRONMENT=development

# MongoDB Configuration
MONGO_URI=mongodb://localhost:27017
DATABASE_NAME=goodpack
```

## 📚 API Endpoints

### Products
- `GET /api/products` - Get all products
- `POST /api/products` - Create a new product
- `GET /api/products/{id}` - Get product by ID
- `PUT /api/products/{id}` - Update product
- `DELETE /api/products/{id}` - Delete product
- `PATCH /api/products/{id}/stock` - Update product stock

### Inventory
- `GET /api/inventory` - Get inventory summary
- `GET /api/categories` - Get all categories

### QR Codes
- `GET /api/qr-codes/{id}` - Get QR code data
- `GET /api/qr-codes/{id}/image` - Download QR code image

### Health
- `GET /api/health` - Health check

## 🗄️ Database Schema

### Products Collection
```json
{
  "_id": "ObjectId",
  "name": "string",
  "description": "string",
  "price": "number",
  "stock": "number",
  "imageUrl": "string (optional)",
  "category": "string (optional)",
  "barcode": "string (optional)",
  "createdAt": "datetime",
  "updatedAt": "datetime"
}
```

## 🏗️ Project Structure

```
goodpack-server/
├── config/          # Configuration management
├── database/        # Database connection
├── handlers/        # HTTP handlers
├── models/          # Data models
├── repository/      # Data access layer
├── routes/          # Route definitions
├── main.go          # Application entry point
├── go.mod           # Go module file
└── README.md        # This file
```

## 🚀 Deployment

### Docker
```bash
# Build image
docker build -t goodpack-server .

# Run container
docker run -p 8080:8080 \
  -e MONGO_URI=mongodb://host.docker.internal:27017 \
  -e DATABASE_NAME=goodpack \
  goodpack-server
```

### Production
1. Set environment variables
2. Use production MongoDB instance
3. Configure reverse proxy (nginx)
4. Enable SSL/TLS
5. Set up monitoring and logging

## 📝 License

MIT License