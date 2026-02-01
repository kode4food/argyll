"""Payment processor step - async payment processing."""

import os
import random
import threading
import time
from datetime import datetime

from argyll import AsyncContext, AttributeType, Client, StepContext, StepResult

# Get engine URL from environment
ENGINE_URL = os.getenv("ARGYLL_ENGINE_URL", "http://localhost:8080")

# Client expects base_url without /engine path prefix
client = Client(ENGINE_URL)


def handle_payment(ctx: StepContext, args: dict) -> StepResult:
    """Process payment asynchronously."""
    order = args.get("order")
    if not isinstance(order, dict):
        return StepResult(success=False, error="order must be an object")

    order_id = order.get("id", "")
    grand_total = order.get("grand_total", 0.0)
    payment_method = order.get("payment_method", "")

    print(
        f"Processing payment (async): order_id={order_id}, "
        f"amount={grand_total}, method={payment_method}"
    )

    # Extract webhook URL from metadata
    webhook_url = ctx.metadata.get("webhook_url")
    if not webhook_url:
        print(f"ERROR: No webhook_url in metadata. Keys: {list(ctx.metadata.keys())}")
        print(f"Metadata: {ctx.metadata}")
        return StepResult(success=False, error="webhook_url not found in metadata")

    print(f"Webhook URL: {webhook_url}")
    async_ctx = AsyncContext(context=ctx, webhook_url=webhook_url)

    def process_payment():
        """Background payment processing."""
        print(
            f"Starting async payment processing: "
            f"order_id={order_id}, flow_id={async_ctx.flow_id}"
        )

        # Simulate processing time (5-10 seconds)
        processing_time = 5 + random.randint(0, 5)
        time.sleep(processing_time)

        # Randomly succeed or fail (50/50)
        success = random.random() < 0.5

        if success:
            payment_result = {
                "order_id": order_id,
                "amount": grand_total,
                "status": "completed",
                "payment_id": f"PAY-{int(time.time())}",
                "processed_at": datetime.utcnow().isoformat() + "Z",
            }

            print(
                f"Payment completed successfully: order_id={order_id}, "
                f"payment_id={payment_result['payment_id']}, "
                f"amount={grand_total}"
            )

            async_ctx.success({"payment_result": payment_result})

        else:
            failure_reasons = [
                "insufficient funds",
                "card declined",
                "expired payment method",
                "fraud detection triggered",
                "payment gateway timeout",
            ]
            reason = random.choice(failure_reasons)

            print(f"Payment failed: order_id={order_id}, reason={reason}")

            async_ctx.fail(f"payment failed: {reason}")

    threading.Thread(target=process_payment, daemon=True).start()

    return StepResult(success=True, outputs={})


if __name__ == "__main__":
    client.new_step("Payment Processor") \
        .with_labels({
            "description": "process payment asynchronously",
            "domain": "payments",
            "capability": "process",
            "execution": "async",
            "example": "true",
        }) \
        .required("order", AttributeType.OBJECT) \
        .output("payment_result", AttributeType.OBJECT) \
        .with_async_execution() \
        .with_timeout(300000) \
        .start(handle_payment)
