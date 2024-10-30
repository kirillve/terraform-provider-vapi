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
