"""Script step example."""

from argyll import Client, AttributeType

client = Client("http://localhost:8080/engine")

# Ale script that doubles a number
script = """
(define (double x) (* x 2))
(double value)
"""

if __name__ == "__main__":
    client.new_step("Double") \
        .required("value", AttributeType.NUMBER) \
        .output("result", AttributeType.NUMBER) \
        .with_script(script) \
        .with_label("category", "math") \
        .register()

    print("Script step 'Double' registered successfully")
