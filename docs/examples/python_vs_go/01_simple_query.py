"""
Example 1: Simple Query
Python Version

This example shows how to send a simple query to Claude and receive responses.
"""

import asyncio
from claude_agent_sdk import Agent


async def main():
    """Send a simple query and print responses"""
    client = Agent()

    prompt = "What is the capital of France?"

    print(f"User: {prompt}\n")

    # Query Claude and stream responses
    async for message in client.query(prompt):
        if message.type == "assistant":
            print(f"Assistant: {message.content}")
        elif message.type == "result":
            print(f"\n✓ Response complete")
            if hasattr(message, 'cost_summary'):
                print(f"  Tokens used - Input: {message.cost_summary.input_tokens}, Output: {message.cost_summary.output_tokens}")


if __name__ == "__main__":
    asyncio.run(main())
