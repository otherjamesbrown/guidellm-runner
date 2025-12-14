#!/usr/bin/env python3
"""Quick test of ai-aas endpoint using guidellm with auth patch."""

import os
import asyncio
import httpx
from guidellm.backends.openai import OpenAIHTTPBackend

# Monkey-patch the backend to include auth headers
original_startup = OpenAIHTTPBackend.process_startup

async def patched_startup(self):
    """Patched startup that includes auth headers."""
    if self._in_process:
        raise RuntimeError("Backend already started up for process.")
    
    api_key = os.environ.get("OPENAI_API_KEY", "")
    headers = {}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"
    
    self._async_client = httpx.AsyncClient(
        http2=self.http2,
        timeout=self.timeout,
        follow_redirects=self.follow_redirects,
        verify=self.verify,
        headers=headers,
        limits=httpx.Limits(
            max_connections=None,
            max_keepalive_connections=None,
            keepalive_expiry=5.0,
        ),
    )
    self._in_process = True

OpenAIHTTPBackend.process_startup = patched_startup

# Now run guidellm
import sys
from guidellm.__main__ import cli
sys.exit(cli())
