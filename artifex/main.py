"""
Project Artifex: AI-Powered Passport Stamp Generation Microservice

This FastAPI service generates stylized digital passport stamps using Replicate's
image generation models (FLUX/SDXL). It implements "Sienna's Creative Logic" to
map arrival context (city, weather, time) to specific visual aesthetics.

Author: TravelMate Team
Created: 2026-02-11
"""

import os
from typing import Literal
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
from dotenv import load_dotenv
import replicate

# Load environment variables
load_dotenv()

# Initialize FastAPI app
app = FastAPI(
    title="Project Artifex",
    description="AI Image Generation Engine for Living Passport Stamps",
    version="1.0.0"
)

# CORS middleware (adjust origins for production)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # TODO: Restrict in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Verify Replicate API token
REPLICATE_API_TOKEN = os.getenv("REPLICATE_API_TOKEN")
if not REPLICATE_API_TOKEN:
    raise RuntimeError("REPLICATE_API_TOKEN environment variable not set")


# ============================================================================
# DATA MODELS
# ============================================================================

class StampRequest(BaseModel):
    """Request model for stamp generation"""
    city: str = Field(..., description="City name (e.g., 'Tokyo', 'Paris')")
    weather: Literal["clear", "rainy", "cloudy"] = Field(
        default="clear",
        description="Weather condition at arrival"
    )
    time_of_day: Literal["morning", "day", "night"] = Field(
        default="day",
        description="Time of day at arrival"
    )
    
    class Config:
        json_schema_extra = {
            "example": {
                "city": "Tokyo",
                "weather": "clear",
                "time_of_day": "morning"
            }
        }


class StampResponse(BaseModel):
    """Response model for generated stamp"""
    status: str
    image_url: str
    prompt_used: str
    mood: str


# ============================================================================
# SIENNA'S CREATIVE LOGIC: Prompt Templates
# ============================================================================

PROMPT_TEMPLATES = {
    "sunny_morning": (
        "A macro shot of a rubber stamp mark on textured paper for {city}. "
        "Ink color is Deep Teal with Golden overlay. Design includes minimalist "
        "line art of {city} landmarks. Crisp edges, high contrast, official "
        "document aesthetic, risograph style."
    ),
    "rainy": (
        "A macro shot of a slightly smudged rubber stamp on damp paper for {city}. "
        "Ink color is Dark Slate and Charcoal. Design includes silhouettes of "
        "{city} streets and rain motifs. Ink bleed into paper fibers, moody, "
        "low contrast, melancholic vibe."
    ),
    "night": (
        "A double-exposure rubber stamp on paper for {city}. Primary ink is "
        "Midnight Blue, secondary misalignment overlay in Neon Cyan. Design "
        "includes abstract city lights and geometry. Electric vibe, chaotic, "
        "vibrant risograph texture."
    )
}


def get_prompt(request: StampRequest) -> tuple[str, str]:
    """
    Determines the appropriate prompt template based on context.
    
    Args:
        request: StampRequest containing city, weather, and time_of_day
        
    Returns:
        Tuple of (formatted_prompt, mood_name)
    """
    # Determine mood based on weather and time
    if request.time_of_day == "night":
        mood = "night"
        template = PROMPT_TEMPLATES["night"]
    elif request.weather in ["rainy", "cloudy"]:
        mood = "rainy"
        template = PROMPT_TEMPLATES["rainy"]
    else:  # clear weather during morning/day
        mood = "sunny_morning"
        template = PROMPT_TEMPLATES["sunny_morning"]
    
    # Format the prompt with city name
    formatted_prompt = template.format(city=request.city)
    
    return formatted_prompt, mood


# ============================================================================
# API ENDPOINTS
# ============================================================================

@app.get("/")
async def root():
    """Health check endpoint"""
    return {
        "status": "Artifex Engine Online",
        "version": "1.0.0",
        "service": "Living Passport Stamp Generator"
    }


@app.get("/health")
async def health_check():
    """Detailed health check"""
    return {
        "status": "healthy",
        "replicate_configured": bool(REPLICATE_API_TOKEN),
        "service": "artifex"
    }


import time
import asyncio
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type

# ... (imports remain the same)

# Custom exception for rate limits
class RateLimitError(Exception):
    pass

@app.post("/generate-stamp", response_model=StampResponse)
async def generate_stamp(request: StampRequest):
    """
    Generate a stylized passport stamp image based on arrival context.
    Includes auto-retry logic for rate limits.
    """
    
    # Define the generation function with retry logic
    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=2, max=10),
        retry=retry_if_exception_type(RateLimitError)
    )
    def call_replicate_with_retry(prompt_text):
        try:
            return replicate.run(
                "black-forest-labs/flux-schnell",
                input={
                    "prompt": prompt_text,
                    "aspect_ratio": "1:1",
                    "output_format": "jpg",
                    "output_quality": 90,
                    "num_outputs": 1
                }
            )
        except replicate.exceptions.ReplicateError as e:
            error_str = str(e).lower()
            if "rate limit" in error_str or "throttled" in error_str:
                print(f"Rate limited. Retrying... ({str(e)})")
                raise RateLimitError("Rate limited by Replicate")
            raise e

    try:
        # Get the appropriate prompt
        prompt, mood = get_prompt(request)
        
        # Call Replicate API with retry logic
        # We wrap the synchronous call in a way that respects the retry decorator
        output = await asyncio.to_thread(call_replicate_with_retry, prompt)
        
        # Extract image URL from output
        if not output or len(output) == 0:
            raise ValueError("No output received from Replicate")
        
        image_url = output[0]
        
        return StampResponse(
            status="success",
            image_url=image_url,
            prompt_used=prompt,
            mood=mood
        )
        
    except RateLimitError:
        raise HTTPException(
            status_code=429,
            detail="Service is currently busy (Rate Limit). Please try again in a few seconds."
        )
    except replicate.exceptions.ReplicateError as e:
        raise HTTPException(
            status_code=500,
            detail=f"Replicate API error: {str(e)}"
        )
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Image generation failed: {str(e)}"
        )


@app.get("/moods")
async def list_moods():
    """List available mood templates"""
    return {
        "moods": list(PROMPT_TEMPLATES.keys()),
        "templates": PROMPT_TEMPLATES
    }


# ============================================================================
# MAIN ENTRY POINT
# ============================================================================

if __name__ == "__main__":
    import uvicorn
    
    port = int(os.getenv("PORT", 8001))
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=port,
        reload=True,  # Enable auto-reload during development
        log_level="info"
    )
