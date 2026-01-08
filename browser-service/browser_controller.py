#!/usr/bin/env python3
"""
Browser Controller for Web3AirdropOS
Manages browser automation via Chrome DevTools Protocol
"""

import asyncio
import json
import os
import websockets
import aiohttp
import redis
from typing import Optional, Dict, Any
import base64
from datetime import datetime

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
CDP_PORT = 9222


class BrowserController:
    def __init__(self):
        self.redis_client = redis.from_url(REDIS_URL)
        self.ws_connection: Optional[websockets.WebSocketClientProtocol] = None
        self.message_id = 0
        self.pending_commands: Dict[int, asyncio.Future] = {}
        
    async def connect_to_browser(self):
        """Connect to Chrome DevTools Protocol"""
        try:
            # Get the debugger URL
            async with aiohttp.ClientSession() as session:
                async with session.get(f"http://localhost:{CDP_PORT}/json") as resp:
                    targets = await resp.json()
                    
            if not targets:
                print("No browser targets found")
                return False
                
            # Connect to first page target
            page_target = next((t for t in targets if t.get("type") == "page"), targets[0])
            ws_url = page_target.get("webSocketDebuggerUrl")
            
            if not ws_url:
                print("No WebSocket URL found")
                return False
                
            self.ws_connection = await websockets.connect(ws_url)
            print(f"Connected to browser: {ws_url}")
            return True
            
        except Exception as e:
            print(f"Failed to connect to browser: {e}")
            return False
    
    async def send_command(self, method: str, params: dict = None) -> dict:
        """Send CDP command and wait for response"""
        if not self.ws_connection:
            raise Exception("Not connected to browser")
            
        self.message_id += 1
        message_id = self.message_id
        
        message = {
            "id": message_id,
            "method": method,
            "params": params or {}
        }
        
        future = asyncio.get_event_loop().create_future()
        self.pending_commands[message_id] = future
        
        await self.ws_connection.send(json.dumps(message))
        
        try:
            result = await asyncio.wait_for(future, timeout=30)
            return result
        except asyncio.TimeoutError:
            del self.pending_commands[message_id]
            raise Exception(f"Command {method} timed out")
    
    async def handle_messages(self):
        """Handle incoming CDP messages"""
        while True:
            try:
                message = await self.ws_connection.recv()
                data = json.loads(message)
                
                # Handle response to our commands
                if "id" in data:
                    message_id = data["id"]
                    if message_id in self.pending_commands:
                        self.pending_commands[message_id].set_result(data)
                        del self.pending_commands[message_id]
                
                # Handle events
                elif "method" in data:
                    await self.handle_event(data)
                    
            except websockets.exceptions.ConnectionClosed:
                print("Browser connection closed")
                break
            except Exception as e:
                print(f"Error handling message: {e}")
    
    async def handle_event(self, event: dict):
        """Handle CDP events"""
        method = event.get("method", "")
        params = event.get("params", {})
        
        # Publish relevant events to Redis
        if method in ["Page.loadEventFired", "Page.frameNavigated", "Network.requestWillBeSent"]:
            self.redis_client.publish("browser:events", json.dumps({
                "type": method,
                "params": params,
                "timestamp": datetime.utcnow().isoformat()
            }))
    
    async def navigate(self, url: str):
        """Navigate to URL"""
        return await self.send_command("Page.navigate", {"url": url})
    
    async def screenshot(self) -> str:
        """Take screenshot and return base64"""
        result = await self.send_command("Page.captureScreenshot", {
            "format": "png",
            "quality": 80
        })
        return result.get("result", {}).get("data", "")
    
    async def click(self, x: int, y: int):
        """Click at coordinates"""
        await self.send_command("Input.dispatchMouseEvent", {
            "type": "mousePressed",
            "x": x,
            "y": y,
            "button": "left",
            "clickCount": 1
        })
        await self.send_command("Input.dispatchMouseEvent", {
            "type": "mouseReleased",
            "x": x,
            "y": y,
            "button": "left",
            "clickCount": 1
        })
    
    async def type_text(self, text: str):
        """Type text"""
        for char in text:
            await self.send_command("Input.dispatchKeyEvent", {
                "type": "keyDown",
                "text": char
            })
            await self.send_command("Input.dispatchKeyEvent", {
                "type": "keyUp",
                "text": char
            })
            await asyncio.sleep(0.05)  # Human-like delay
    
    async def evaluate(self, expression: str) -> Any:
        """Evaluate JavaScript"""
        result = await self.send_command("Runtime.evaluate", {
            "expression": expression,
            "returnByValue": True
        })
        return result.get("result", {}).get("result", {}).get("value")
    
    async def listen_for_commands(self):
        """Listen for commands from Redis"""
        pubsub = self.redis_client.pubsub()
        pubsub.subscribe("browser:commands")
        
        print("Listening for browser commands...")
        
        for message in pubsub.listen():
            if message["type"] == "message":
                try:
                    command = json.loads(message["data"])
                    await self.process_command(command)
                except Exception as e:
                    print(f"Error processing command: {e}")
    
    async def process_command(self, command: dict):
        """Process a command from Redis"""
        action = command.get("action")
        session_id = command.get("session_id")
        
        result = {"session_id": session_id, "success": False}
        
        try:
            if action == "navigate":
                await self.navigate(command.get("url"))
                result["success"] = True
                
            elif action == "screenshot":
                data = await self.screenshot()
                result["success"] = True
                result["data"] = data
                
            elif action == "click":
                await self.click(command.get("x", 0), command.get("y", 0))
                result["success"] = True
                
            elif action == "type":
                await self.type_text(command.get("text", ""))
                result["success"] = True
                
            elif action == "evaluate":
                value = await self.evaluate(command.get("expression", ""))
                result["success"] = True
                result["value"] = value
                
            elif action == "get_url":
                url = await self.evaluate("window.location.href")
                result["success"] = True
                result["url"] = url
                
        except Exception as e:
            result["error"] = str(e)
        
        # Publish result
        self.redis_client.publish(f"browser:result:{session_id}", json.dumps(result))
    
    async def run(self):
        """Main run loop"""
        # Wait for browser to start
        await asyncio.sleep(5)
        
        # Connect to browser
        while True:
            if await self.connect_to_browser():
                break
            print("Retrying connection in 5 seconds...")
            await asyncio.sleep(5)
        
        # Enable required domains
        await self.send_command("Page.enable")
        await self.send_command("Network.enable")
        await self.send_command("Runtime.enable")
        
        # Run message handler and command listener concurrently
        await asyncio.gather(
            self.handle_messages(),
            self.listen_for_commands()
        )


if __name__ == "__main__":
    controller = BrowserController()
    asyncio.run(controller.run())
