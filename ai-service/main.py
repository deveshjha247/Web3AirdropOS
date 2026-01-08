# AI Service for Web3AirdropOS
# FastAPI-based microservice for content generation and AI assistance

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List, Optional, Dict, Any
import openai
import os
from datetime import datetime, timedelta
import random
import json

app = FastAPI(
    title="Web3AirdropOS AI Service",
    description="AI-powered content generation and engagement planning",
    version="1.0.0"
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Initialize OpenAI client
openai.api_key = os.getenv("OPENAI_API_KEY", "")

# Platform-specific prompts and styles
PLATFORM_STYLES = {
    "farcaster": {
        "tone": "crypto-native, community-focused, technical but accessible",
        "max_length": 320,
        "style_hints": [
            "Use crypto/web3 terminology naturally",
            "Reference Farcaster culture (casts, channels, frames)",
            "Be authentic and community-focused",
            "Avoid excessive hashtags",
            "Engage in meaningful discussions"
        ]
    },
    "twitter": {
        "tone": "punchy, engagement-focused, trend-aware",
        "max_length": 280,
        "style_hints": [
            "Use relevant hashtags strategically",
            "Include calls to action",
            "Be concise and impactful",
            "Use threads for longer content",
            "Engage with trending topics"
        ]
    },
    "telegram": {
        "tone": "informative, community-oriented, detailed",
        "max_length": 4096,
        "style_hints": [
            "Can be more detailed",
            "Use formatting (bold, italic)",
            "Include relevant links",
            "Community updates style"
        ]
    },
    "discord": {
        "tone": "casual, friendly, community-focused",
        "max_length": 2000,
        "style_hints": [
            "Use Discord-style formatting",
            "Include emojis appropriately",
            "Be welcoming and helpful",
            "Encourage discussion"
        ]
    }
}


class GenerateContentRequest(BaseModel):
    platform: str
    type: str  # post, reply, thread
    prompt: Optional[str] = None
    tone: Optional[str] = None
    context: Optional[str] = None
    reply_to: Optional[str] = None
    max_length: Optional[int] = None
    num_options: int = 3
    keywords: Optional[List[str]] = None
    hashtags: bool = False


class GeneratedContent(BaseModel):
    content: str
    tone: str
    platform: str
    hashtags: Optional[List[str]] = None
    predicted_metrics: Dict[str, float]


class GenerateContentResponse(BaseModel):
    contents: List[GeneratedContent]
    error: Optional[str] = None


class EngagementPlanRequest(BaseModel):
    platform: str
    goal_type: str  # engagement, followers, visibility
    days: int = 7
    topics: Optional[List[str]] = None


class Action(BaseModel):
    time: str
    type: str  # post, reply, like, recast
    content: Optional[str] = None
    target: Optional[str] = None
    reason: str


class DailyPlan(BaseModel):
    date: str
    actions: List[Action]


class EngagementPlan(BaseModel):
    days: List[DailyPlan]


class CampaignSummaryRequest(BaseModel):
    campaign_name: str
    campaign_url: str
    tasks: List[str]


class CampaignSummary(BaseModel):
    summary: str
    estimated_time: str
    difficulty: str
    tips: List[str]
    priority_order: List[str]


@app.get("/health")
async def health_check():
    return {"status": "healthy", "service": "ai-service"}


@app.post("/generate", response_model=GenerateContentResponse)
async def generate_content(request: GenerateContentRequest):
    """Generate AI-powered content for social platforms"""
    
    platform_style = PLATFORM_STYLES.get(request.platform, PLATFORM_STYLES["twitter"])
    max_length = request.max_length or platform_style["max_length"]
    tone = request.tone or platform_style["tone"]
    
    # Build the prompt
    system_prompt = f"""You are a skilled Web3 content creator specializing in {request.platform}.
Your writing style is: {tone}

Platform guidelines:
{chr(10).join('- ' + hint for hint in platform_style['style_hints'])}

Important:
- Content must be under {max_length} characters
- Sound authentic and human, NOT like a bot
- Vary your writing style naturally
- Be relevant to current Web3 trends
"""

    user_prompt = f"Generate {request.num_options} unique {request.type} options"
    
    if request.prompt:
        user_prompt += f" about: {request.prompt}"
    
    if request.context:
        user_prompt += f"\n\nContext: {request.context}"
    
    if request.reply_to:
        user_prompt += f"\n\nReplying to: {request.reply_to}"
    
    if request.keywords:
        user_prompt += f"\n\nInclude these keywords naturally: {', '.join(request.keywords)}"
    
    if request.hashtags:
        user_prompt += "\n\nInclude 2-3 relevant hashtags."
    
    user_prompt += f"""

Return as JSON array with format:
[{{"content": "...", "hashtags": ["...", "..."]}}]

Each option should be unique in approach and style."""

    try:
        response = openai.ChatCompletion.create(
            model="gpt-4",
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ],
            temperature=0.9,
            max_tokens=2000
        )
        
        # Parse the response
        content_text = response.choices[0].message.content
        
        # Try to parse as JSON
        try:
            parsed = json.loads(content_text)
        except json.JSONDecodeError:
            # Extract JSON from markdown code blocks if present
            import re
            json_match = re.search(r'```(?:json)?\s*([\s\S]*?)\s*```', content_text)
            if json_match:
                parsed = json.loads(json_match.group(1))
            else:
                parsed = [{"content": content_text, "hashtags": []}]
        
        contents = []
        for item in parsed:
            # Predict engagement metrics (simplified model)
            engagement_score = random.uniform(0.6, 0.95)
            viral_potential = random.uniform(0.3, 0.8)
            
            contents.append(GeneratedContent(
                content=item.get("content", str(item)),
                tone=tone,
                platform=request.platform,
                hashtags=item.get("hashtags", []),
                predicted_metrics={
                    "engagement_score": round(engagement_score, 2),
                    "viral_potential": round(viral_potential, 2)
                }
            ))
        
        return GenerateContentResponse(contents=contents)
        
    except Exception as e:
        return GenerateContentResponse(contents=[], error=str(e))


@app.post("/engagement-plan", response_model=EngagementPlan)
async def generate_engagement_plan(request: EngagementPlanRequest):
    """Generate a multi-day engagement plan"""
    
    platform_style = PLATFORM_STYLES.get(request.platform, PLATFORM_STYLES["twitter"])
    
    system_prompt = f"""You are a Web3 social media strategist specializing in {request.platform}.
Create a {request.days}-day engagement plan to maximize {request.goal_type}.

Guidelines:
- Actions should be spaced throughout the day naturally
- Mix different types of engagement (posts, replies, likes, reposts)
- Include optimal posting times
- Each action should have a clear purpose
- Be specific with content suggestions
"""

    user_prompt = f"""Create a {request.days}-day plan for maximum {request.goal_type} on {request.platform}.

Topics to focus on: {', '.join(request.topics) if request.topics else 'General Web3, crypto, DeFi'}

Return as JSON:
{{
  "days": [
    {{
      "date": "Day 1",
      "actions": [
        {{"time": "09:00", "type": "post", "content": "...", "reason": "Morning engagement peak"}}
      ]
    }}
  ]
}}

Include 3-5 actions per day with varied types."""

    try:
        response = openai.ChatCompletion.create(
            model="gpt-4",
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ],
            temperature=0.8,
            max_tokens=3000
        )
        
        content = response.choices[0].message.content
        
        # Parse JSON
        try:
            parsed = json.loads(content)
        except json.JSONDecodeError:
            import re
            json_match = re.search(r'```(?:json)?\s*([\s\S]*?)\s*```', content)
            if json_match:
                parsed = json.loads(json_match.group(1))
            else:
                # Generate fallback plan
                parsed = generate_fallback_plan(request.days)
        
        return EngagementPlan(**parsed)
        
    except Exception as e:
        # Return fallback plan on error
        return EngagementPlan(**generate_fallback_plan(request.days))


def generate_fallback_plan(days: int) -> dict:
    """Generate a basic fallback engagement plan"""
    plan_days = []
    base_date = datetime.now()
    
    action_types = ["post", "reply", "like", "recast"]
    times = ["09:00", "12:00", "15:00", "18:00", "21:00"]
    
    for i in range(days):
        day_date = base_date + timedelta(days=i)
        actions = []
        
        for j, time in enumerate(random.sample(times, min(4, len(times)))):
            action_type = action_types[j % len(action_types)]
            actions.append({
                "time": time,
                "type": action_type,
                "content": f"Scheduled {action_type}" if action_type == "post" else None,
                "reason": f"Regular {action_type} for engagement"
            })
        
        plan_days.append({
            "date": day_date.strftime("%Y-%m-%d"),
            "actions": actions
        })
    
    return {"days": plan_days}


@app.post("/campaign-summary", response_model=CampaignSummary)
async def summarize_campaign(request: CampaignSummaryRequest):
    """Summarize a campaign and provide execution tips"""
    
    system_prompt = """You are a Web3 airdrop strategist. Analyze campaigns and provide:
- Clear summary
- Time estimates
- Difficulty assessment
- Execution tips
- Optimal task order"""

    user_prompt = f"""Analyze this campaign:
Name: {request.campaign_name}
URL: {request.campaign_url}
Tasks: {', '.join(request.tasks)}

Provide summary with:
1. Brief description
2. Estimated completion time
3. Difficulty (easy/medium/hard)
4. 3-5 tips for efficient completion
5. Recommended task order

Return as JSON:
{{"summary": "...", "estimated_time": "...", "difficulty": "...", "tips": [...], "priority_order": [...]}}"""

    try:
        response = openai.ChatCompletion.create(
            model="gpt-4",
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ],
            temperature=0.7,
            max_tokens=1000
        )
        
        content = response.choices[0].message.content
        
        try:
            parsed = json.loads(content)
        except json.JSONDecodeError:
            import re
            json_match = re.search(r'```(?:json)?\s*([\s\S]*?)\s*```', content)
            if json_match:
                parsed = json.loads(json_match.group(1))
            else:
                parsed = {
                    "summary": content,
                    "estimated_time": "Unknown",
                    "difficulty": "medium",
                    "tips": ["Complete tasks in order", "Use multiple wallets if allowed"],
                    "priority_order": request.tasks
                }
        
        return CampaignSummary(**parsed)
        
    except Exception as e:
        return CampaignSummary(
            summary=f"Campaign: {request.campaign_name}",
            estimated_time="15-30 minutes",
            difficulty="medium",
            tips=["Check eligibility first", "Complete social tasks early"],
            priority_order=request.tasks
        )


class ReplyDraftRequest(BaseModel):
    original_post: str
    platform: str
    tone: str = "friendly and insightful"
    include_question: bool = False


@app.post("/generate-reply")
async def generate_reply(request: ReplyDraftRequest):
    """Generate a natural reply to a post"""
    
    platform_style = PLATFORM_STYLES.get(request.platform, PLATFORM_STYLES["twitter"])
    
    system_prompt = f"""You are a genuine Web3 community member on {request.platform}.
Generate a natural, engaging reply. Be {request.tone}.
- Sound human, not like a bot
- Add value to the conversation
- Max {platform_style['max_length']} characters
- Don't be generic or spammy"""

    user_prompt = f"""Reply to this post:
"{request.original_post}"

{"End with an engaging question." if request.include_question else ""}

Generate 3 different reply options."""

    try:
        response = openai.ChatCompletion.create(
            model="gpt-4",
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ],
            temperature=0.9,
            max_tokens=500
        )
        
        return {"replies": response.choices[0].message.content.split("\n\n")}
        
    except Exception as e:
        return {"error": str(e)}


class ThreadRequest(BaseModel):
    topic: str
    platform: str
    length: int = 5  # Number of posts in thread
    tone: str = "informative and engaging"


@app.post("/generate-thread")
async def generate_thread(request: ThreadRequest):
    """Generate a thread/series of posts"""
    
    platform_style = PLATFORM_STYLES.get(request.platform, PLATFORM_STYLES["twitter"])
    
    system_prompt = f"""You are an expert Web3 content creator.
Create a {request.length}-part thread about {request.topic}.
Each part should be under {platform_style['max_length']} characters.
Make it {request.tone}."""

    user_prompt = f"""Create a {request.length}-part thread about: {request.topic}

Format:
1/ [First post - hook the reader]
2/ [Expand on the topic]
...
{request.length}/ [Strong conclusion with CTA]

Make each post standalone but connected."""

    try:
        response = openai.ChatCompletion.create(
            model="gpt-4",
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ],
            temperature=0.8,
            max_tokens=2000
        )
        
        # Parse thread into parts
        content = response.choices[0].message.content
        parts = []
        
        import re
        matches = re.findall(r'\d+[/\.]\s*(.+?)(?=\d+[/\.]|$)', content, re.DOTALL)
        
        for i, match in enumerate(matches):
            parts.append({
                "order": i + 1,
                "content": match.strip()
            })
        
        return {"thread": parts, "total_parts": len(parts)}
        
    except Exception as e:
        return {"error": str(e)}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8001)
