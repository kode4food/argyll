"""Script step example."""

from argyll import Client, AttributeType

client = Client("http://localhost:8080/engine")

if __name__ == "__main__":
    # Ale script that doubles a number
    client.new_step("Double") \
        .required("value", AttributeType.NUMBER) \
        .output("result", AttributeType.NUMBER) \
        .with_script("(* value 2)") \
        .with_label("category", "math") \
        .register()

    print("Script step 'Double' registered successfully")
