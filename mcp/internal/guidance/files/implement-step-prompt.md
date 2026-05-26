Implement an Argyll {{language}} step named {{step_name}}.

Requirements:
{{requirements}}

Use the Argyll SDK guidance: prefer SDK-hosted Start/start for POST handlers; use Register/register with WithEndpoint/with_endpoint and WithMethod/with_method when a sync or async step is backed by an existing GET, PUT, or DELETE endpoint. Declare inputs and outputs explicitly and use async context/webhooks for async steps.
