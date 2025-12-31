"""
Example 2: Permission Control
Python Version

This example demonstrates how to control which tools Claude can use
by implementing a permission callback.
"""

import asyncio
from claude_agent_sdk import Agent, PermissionBehavior


async def check_permission(tool_name: str, input_data: dict, context) -> str:
    """
    Permission callback that controls tool access.
    Returns "allow" or "deny" to grant or restrict tool use.
    """
    print(f"🔐 Checking permission for tool: {tool_name}")

    # Block dangerous bash commands
    if tool_name == "Bash":
        command = input_data.get("command", "")
        dangerous_commands = ["rm -rf", "dd if=/dev/zero", ":(){ :|:& };:"]

        for dangerous in dangerous_commands:
            if dangerous in command:
                print(f"❌ Blocked dangerous command: {dangerous}")
                return PermissionBehavior.DENY

    print(f"✅ Permission granted for {tool_name}")
    return PermissionBehavior.ALLOW


async def main():
    """
    Query Claude with permission control.
    Claude will ask before using restricted tools.
    """
    client = Agent(
        can_use_tool=check_permission,
        allowed_tools=["Bash", "Write", "Read"],
        system_prompt="You are a helpful assistant. You have access to tools but must ask before using them."
    )

    prompt = "Create a test file at /tmp/test.txt and list its contents"
    print(f"User: {prompt}\n")

    async for message in client.query(prompt):
        if message.type == "assistant":
            print(f"Assistant: {message.content}")


if __name__ == "__main__":
    asyncio.run(main())
