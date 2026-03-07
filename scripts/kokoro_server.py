"""
Kokoro TTS server — OpenAI-compatible /v1/audio/speech endpoint.
Listens on port 5000.

Setup:
    pip install -r requirements.txt

Model files (download once into the receptionist/ directory):
    curl -L https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/kokoro-v1.0.onnx -o kokoro-v1.0.onnx
    curl -L https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/voices-v1.0.bin  -o voices-v1.0.bin
"""

import io
import os
import sys

import numpy as np
import soundfile as sf
from fastapi import FastAPI, HTTPException
from fastapi.responses import Response
from pydantic import BaseModel

try:
    from kokoro_onnx import Kokoro
except ImportError:
    print("ERROR: kokoro_onnx not installed. Run: pip install -r requirements.txt")
    sys.exit(1)

MODEL_PATH  = os.getenv("KOKORO_MODEL",  "kokoro-v1.0.onnx")
VOICES_PATH = os.getenv("KOKORO_VOICES", "voices-v1.0.bin")

print(f"Loading Kokoro from {MODEL_PATH} / {VOICES_PATH} ...")
kokoro = Kokoro(MODEL_PATH, VOICES_PATH)
print("Kokoro ready.")

app = FastAPI(title="Kokoro TTS")


class SpeechRequest(BaseModel):
    input: str
    voice: str = "af_sky"
    response_format: str = "wav"  # only wav supported
    speed: float = 1.0


@app.post("/v1/audio/speech")
async def speech(req: SpeechRequest):
    if not req.input.strip():
        raise HTTPException(status_code=400, detail="input is empty")
    try:
        samples, sr = kokoro.create(
            req.input,
            voice=req.voice,
            speed=req.speed,
            lang="en-us",
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

    buf = io.BytesIO()
    sf.write(buf, samples, sr, format="WAV", subtype="PCM_16")
    buf.seek(0)
    return Response(content=buf.read(), media_type="audio/wav")


@app.get("/health")
def health():
    return {"ok": True}


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", 5000))
    uvicorn.run(app, host="0.0.0.0", port=port)
