"""Script step example."""

from argyll import Client, AttributeType, ScriptConfig, ScriptLanguage

client = Client("http://localhost:8080")

if __name__ == "__main__":
    # Lua script that doubles a number
    client.new_step().with_name("Double") \
        .required("value", AttributeType.NUMBER) \
        .output("result", AttributeType.NUMBER) \
        .with_script(ScriptConfig(
            language=ScriptLanguage.LUA,
            script="return {result = value * 2}",
        )) \
        .with_label("category", "math") \
        .register()

    print("Script step 'Double' registered successfully")
