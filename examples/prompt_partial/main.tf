terraform {
  required_providers {
    portkey = {
      source = "registry.terraform.io/portkey-ai/portkey"
    }
  }
}

provider "portkey" {}

# Create a reusable prompt partial
resource "portkey_prompt_partial" "system_instructions" {
  name    = "System Instructions"
  content = "You are a helpful AI assistant. Always be concise and professional."
}

# Create another partial for common formatting rules
resource "portkey_prompt_partial" "formatting_rules" {
  name    = "Formatting Rules"
  content = "Format your response as markdown. Use bullet points for lists. Keep paragraphs short."
}

# Look up an existing partial
data "portkey_prompt_partial" "existing" {
  slug = portkey_prompt_partial.system_instructions.slug
}

# List all partials
data "portkey_prompt_partials" "all" {}

output "system_instructions_slug" {
  value = portkey_prompt_partial.system_instructions.slug
}

output "formatting_rules_slug" {
  value = portkey_prompt_partial.formatting_rules.slug
}

output "all_partials" {
  value = [for p in data.portkey_prompt_partials.all.prompt_partials : p.name]
}
