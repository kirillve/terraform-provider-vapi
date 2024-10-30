---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vapi_assistant Resource - vapi"
subcategory: ""
description: |-
  Manages an assistant resource in the VAPI system.
---

# vapi_assistant (Resource)

Manages an assistant resource in the VAPI system.

## Example Usage

```terraform
resource "vapi_assistant" "example_assistant" {
  name               = "example-assistant"
  first_message_mode = "assistant-speaks-first"
  hipaa_enabled      = false

  server_url        = "https://somewhere.com"
  server_url_secret = "secret-phrase"

  client_messages = [
    "function-call",
    "hang",
    "status-update",
    "tool-calls",
    "user-interrupted",
  ]

  server_messages = [
    "end-of-call-report",
    "function-call",
    "hang",
    "status-update",
    "tool-calls",
    "user-interrupted",
    "transfer-destination-request",
  ]

  silence_timeout_seconds = 30
  max_duration_seconds    = 600
  background_sound        = "office"
  background_denoising    = true
  model_output_enabled    = true

  first_message     = "First message"
  voicemail_message = "Voicemain message"
  end_call_message  = "End Call message"
  end_call_phrases = [
    "Bye for now",
    "Talk to you soon",
    "Goodbye",
    "Talk to you later",
  ]

  transcriber = {
    provider = "deepgram"
    model    = "nova-2-phonecall"
    language = "en"
    keywords = []
  }

  model = {
    model         = "gpt-4o"
    system_prompt = "System prompt"
    temperature   = 0.3
    provider      = "openai"
  }

  voice = {
    model            = "eleven_turbo_v2_5"
    provider         = "11labs"
    voice_id         = "FVQMzxJGPUBtfz1Azdoy"
    stability        = 1.0
    similarity_boost = 0.75
  }

  start_speaking_plan = {
    wait_seconds              = 2.5
    smart_endpointing_enabled = true
    transcription_endpointing_plan = {
      on_punctuation_seconds    = 0.5
      on_no_punctuation_seconds = 1.0
      on_number_seconds         = 0.75
    }
  }

  stop_speaking_plan = {
    num_words = 3
  }

  analysis_plan = {
    summary_prompt         = "Summary prompt"
    structured_data_prompt = "Structured data prompt"
    structured_data_schema = {
      type = "object"
      properties = {
        isCallNotAnswered = {
          description = "Indicates whether the call was not answered by the customer.",
          type        = "string"
        }
      }
    }
    success_evaluation_prompt = "Success evaluation prompt"
    success_evaluation_rubric = "Checklist"
  }

  message_plan = {
    idle_messages = [
      "I’ll proceed with scheduling unless you have any questions.",
      "If you don't mind, I will continue with booking your appointment?",
      "Sorry, I kind of lost track for a moment. Mind if we continued?",
      "I will continue, but feel free to ask me any questions, ok?",
      "Are you ready to proceed?"
    ]
  }

  end_call_function_enabled = false
  recording_enabled         = true
  forwarding_phone_number   = ""
  phone_number_id           = ""
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the assistant resource.

### Optional

- `analysis_plan` (Attributes) Configuration for the analysis plan. (see [below for nested schema](#nestedatt--analysis_plan))
- `background_denoising` (Boolean) Indicates whether background denoising is enabled.
- `background_sound` (String) Background sound used during the call.
- `client_messages` (List of String) List of messages from the client.
- `end_call_function_enabled` (Boolean) Enables the end call function.
- `end_call_message` (String) Message to be used when ending the call.
- `end_call_phrases` (List of String) List of phrases to end the call.
- `first_message` (String) Initial message sent to the client.
- `first_message_mode` (String) Mode of the first message.
- `forwarding_phone_number` (String) Phone number to which calls are forwarded.
- `hipaa_enabled` (Boolean) Indicates whether HIPAA compliance is enabled.
- `max_duration_seconds` (Number) Maximum duration of the call in seconds.
- `message_plan` (Attributes) Configuration for message plan. (see [below for nested schema](#nestedatt--message_plan))
- `model` (Attributes) Configuration for the assistant model. (see [below for nested schema](#nestedatt--model))
- `model_output_enabled` (Boolean) Indicates whether model output is enabled.
- `phone_number_id` (String) ID of the phone number associated with the assistant.
- `recording_enabled` (Boolean) Indicates if call recording is enabled.
- `server_messages` (List of String) List of messages from the server.
- `server_url` (String) Server URL.
- `server_url_secret` (String) Server URL Secret.
- `silence_timeout_seconds` (Number) Timeout in seconds for silence.
- `start_speaking_plan` (Attributes) Configuration for starting the speaking plan. (see [below for nested schema](#nestedatt--start_speaking_plan))
- `stop_speaking_plan` (Attributes) Configuration for stopping the speaking plan. (see [below for nested schema](#nestedatt--stop_speaking_plan))
- `transcriber` (Attributes) Configuration for the transcriber model. (see [below for nested schema](#nestedatt--transcriber))
- `voice` (Attributes) Configuration for the voice model. (see [below for nested schema](#nestedatt--voice))
- `voicemail_message` (String) Message to be used for voicemail.

### Read-Only

- `id` (String) Unique identifier for the assistant resource.

<a id="nestedatt--analysis_plan"></a>
### Nested Schema for `analysis_plan`

Optional:

- `structured_data_prompt` (String) Prompt for structured data generation.
- `structured_data_schema` (Attributes) Schema for structured data. (see [below for nested schema](#nestedatt--analysis_plan--structured_data_schema))
- `success_evaluation_prompt` (String) Prompt for success evaluation.
- `success_evaluation_rubric` (String) Rubric for evaluating success.
- `summary_prompt` (String) Prompt for generating a summary.

<a id="nestedatt--analysis_plan--structured_data_schema"></a>
### Nested Schema for `analysis_plan.structured_data_schema`

Optional:

- `properties` (Attributes Map) Properties of structured data fields. (see [below for nested schema](#nestedatt--analysis_plan--structured_data_schema--properties))
- `type` (String) Type of structured data.

<a id="nestedatt--analysis_plan--structured_data_schema--properties"></a>
### Nested Schema for `analysis_plan.structured_data_schema.properties`

Optional:

- `description` (String) Description of the property.
- `type` (String) Type of the property.




<a id="nestedatt--message_plan"></a>
### Nested Schema for `message_plan`

Optional:

- `idle_messages` (List of String) List of idle messages.


<a id="nestedatt--model"></a>
### Nested Schema for `model`

Required:

- `model` (String) The assistant model type.

Optional:

- `provider` (String) Provider for the assistant model: openai, azure-openai, together-ai, anyscale, openrouter, perplexity-ai, deepinfra, custom-llm, runpod, groq, vapi, anthropic, google
- `system_prompt` (String) Prompt text used to guide the assistant model.
- `temperature` (Number) Temperature setting for the model's response randomness.
- `tool_ids` (List of String) List of tool IDs used by the model.


<a id="nestedatt--start_speaking_plan"></a>
### Nested Schema for `start_speaking_plan`

Optional:

- `smart_endpointing_enabled` (Boolean) Enable smart endpointing for better control.
- `transcription_endpointing_plan` (Attributes) Endpointing plan for transcription. (see [below for nested schema](#nestedatt--start_speaking_plan--transcription_endpointing_plan))
- `wait_seconds` (Number) Seconds to wait before speaking starts.

<a id="nestedatt--start_speaking_plan--transcription_endpointing_plan"></a>
### Nested Schema for `start_speaking_plan.transcription_endpointing_plan`

Optional:

- `on_no_punctuation_seconds` (Number) Delay in seconds for no punctuation.
- `on_number_seconds` (Number) Delay in seconds for number-based endpointing.
- `on_punctuation_seconds` (Number) Delay in seconds for punctuation-based endpointing.



<a id="nestedatt--stop_speaking_plan"></a>
### Nested Schema for `stop_speaking_plan`

Optional:

- `backoff_seconds` (Number) Backoff period in seconds before stopping.
- `num_words` (Number) Number of words required to stop speaking.
- `voice_seconds` (Number) Duration in seconds to stop speaking.


<a id="nestedatt--transcriber"></a>
### Nested Schema for `transcriber`

Required:

- `provider` (String) Provider for the transcriber service.

Optional:

- `keywords` (List of String) List of keywords to focus on during transcription.
- `model` (String) Model used for transcription.


<a id="nestedatt--voice"></a>
### Nested Schema for `voice`

Required:

- `model` (String) Model for the voice model.
- `provider` (String) Provider for the voice model.
- `voice_id` (String) ID of the voice model.

Optional:

- `similarity_boost` (Number) Boost factor for similarity in voice.
- `stability` (Number) Stability of the voice output.