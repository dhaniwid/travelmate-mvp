# Project Artifex

AI-Powered Passport Stamp Generation Microservice for TravelMate's "Living Passport" feature.

## Overview

Artifex is a FastAPI-based microservice that generates stylized digital passport stamps using Replicate's AI image generation models (FLUX Schnell/SDXL). It implements "Sienna's Creative Logic" to map arrival context (city, weather, time) to specific visual aesthetics.

## Architecture

```
artifex/
├── main.py              # FastAPI application
├── requirements.txt     # Python dependencies
├── .env.example        # Environment template
└── README.md           # This file
```

## Setup

### 1. Install Dependencies

```bash
cd artifex
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env and add your Replicate API token
```

Get your Replicate API token from: https://replicate.com/account/api-tokens

### 3. Run the Service

```bash
python main.py
```

The service will start on `http://localhost:8001`

## API Endpoints

### `GET /`
Health check endpoint.

**Response:**
```json
{
  "status": "Artifex Engine Online",
  "version": "1.0.0",
  "service": "Living Passport Stamp Generator"
}
```

### `POST /generate-stamp`
Generate a passport stamp based on arrival context.

**Request Body:**
```json
{
  "city": "Tokyo",
  "weather": "clear",
  "time_of_day": "morning"
}
```

**Parameters:**
- `city` (string, required): City name (e.g., "Tokyo", "Paris")
- `weather` (string): One of `"clear"`, `"rainy"`, `"cloudy"` (default: `"clear"`)
- `time_of_day` (string): One of `"morning"`, `"day"`, `"night"` (default: `"day"`)

**Response:**
```json
{
  "status": "success",
  "image_url": "https://replicate.delivery/...",
  "prompt_used": "A macro shot of a rubber stamp...",
  "mood": "sunny_morning"
}
```

### `GET /moods`
List available mood templates.

**Response:**
```json
{
  "moods": ["sunny_morning", "rainy", "night"],
  "templates": { ... }
}
```

## Mood Logic

The service maps arrival context to three distinct visual moods:

| Context | Mood | Visual Style |
|---------|------|--------------|
| **Morning/Day + Clear** | `sunny_morning` | Deep Teal + Golden overlay, crisp edges, high contrast |
| **Any Time + Rain/Cloudy** | `rainy` | Dark Slate + Charcoal, ink bleed, moody, melancholic |
| **Night** | `night` | Midnight Blue + Neon Cyan, double-exposure, electric vibe |

## Integration with TravelMate Backend

### From Go Backend

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
)

type StampRequest struct {
    City      string `json:"city"`
    Weather   string `json:"weather"`
    TimeOfDay string `json:"time_of_day"`
}

func generateStamp(city, weather, timeOfDay string) (string, error) {
    reqBody, _ := json.Marshal(StampRequest{
        City:      city,
        Weather:   weather,
        TimeOfDay: timeOfDay,
    })
    
    resp, err := http.Post(
        "http://localhost:8001/generate-stamp",
        "application/json",
        bytes.NewBuffer(reqBody),
    )
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    
    return result["image_url"].(string), nil
}
```

### From Next.js Frontend

```typescript
interface StampRequest {
  city: string;
  weather: 'clear' | 'rainy' | 'cloudy';
  time_of_day: 'morning' | 'day' | 'night';
}

async function generateStamp(request: StampRequest): Promise<string> {
  const response = await fetch('http://localhost:8001/generate-stamp', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });
  
  const data = await response.json();
  return data.image_url;
}
```

## Development

### Running with Auto-Reload

```bash
uvicorn main:app --reload --port 8001
```

### Testing the API

```bash
# Health check
curl http://localhost:8001/

# Generate a stamp
curl -X POST http://localhost:8001/generate-stamp \
  -H "Content-Type: application/json" \
  -d '{"city": "Tokyo", "weather": "clear", "time_of_day": "morning"}'
```

## Production Deployment

### Using Docker

Create `Dockerfile`:
```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8001"]
```

Build and run:
```bash
docker build -t artifex .
docker run -p 8001:8001 --env-file .env artifex
```

### Environment Variables

- `REPLICATE_API_TOKEN` (required): Your Replicate API token
- `PORT` (optional): Port to run the service on (default: 8001)

## Future Enhancements

- [ ] Add caching layer (Redis) for generated stamps
- [ ] Implement city-specific landmark configuration
- [ ] Add support for custom color palettes per city
- [ ] Implement rate limiting
- [ ] Add image post-processing (watermarks, compression)
- [ ] Support for batch generation

## Related Documentation

- [Digital Passport Engine Spec](../travelmate-web/docs/specs/digital_passport_engine.md)
- [Passport Stamp Design Guide](../brain/passport_stamps_hybrid_recommended.md)

## License

Proprietary - TravelMate 2026
