"""Async step example."""

import threading
import time

from argyll import Client, StepContext, AsyncContext, AttributeType, StepResult

client = Client("http://localhost:8080")


def handle_async_task(ctx: StepContext, args: dict) -> StepResult:
    """Async task that processes in background."""
    webhook_url = ctx.metadata.get("webhook_url")
    if not webhook_url:
        return StepResult(success=False, error="No webhook URL")

    async_ctx = AsyncContext(context=ctx, webhook_url=webhook_url)
    duration = args.get("duration", 5)

    def process():
        """Background processing."""
        try:
            print(f"Starting async work for {duration} seconds...")
            time.sleep(duration)
            result = {"status": "completed", "duration": duration}
            print(f"Async work completed: {result}")
            async_ctx.success(result)
        except Exception as e:
            print(f"Async work failed: {e}")
            async_ctx.fail(str(e))

    threading.Thread(target=process, daemon=True).start()

    return StepResult(success=True, outputs={"status": "started"})


if __name__ == "__main__":
    client.new_step("AsyncTask") \
        .with_async_execution() \
        .optional("duration", AttributeType.NUMBER, "5") \
        .output("status", AttributeType.STRING) \
        .output("duration", AttributeType.NUMBER) \
        .start(handle_async_task)
