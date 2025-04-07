
# Setup Instructions

## Installation

Provide detailed steps on how to install the application. This includes using the `make install` command.

-   Clone the repository
-   Head over to groq.com and sign up for a dev account and generate an API key.
-   Run `make install`
-   Any specific system requirements

## Configuration
1.  Sample configuration is available at samples/recipes.

    Template for `config.yaml` structure:

    ```yaml
    # Example configuration
    name: "media_publisher" #workspace_name
    scope: "content_creation" #workspace_creation
    planner: #planner definition
        model: "deepseek-r1-distill-llama-70b"
        provider: "groq"
    orchestrator: # orchestrator definition
        name: "editor_in_chief"
        description: "Oversees content planning, publishing flow, and quality checks."
        scope: "Help user plan, create, edit, and publish media content efficiently."
        model: "llama-3.1-8b-instant"
        provider: "groq"
        goal: "Streamline end-to-end media content publishing."
        team: #team of agents
            content_creator:
                name: "content_creator"
                capabilities: "generate articles, scripts, and media content"
                description: "Creates engaging content for publishing."
                model: "llama-3.2-1b-preview"
                provider: "groq"
                goal: "Produce high-quality media-ready content."
                depends: "editor_in_chief"
                responds: "editor_in_chief"
                status: "Initialised"
            content_editor:
                name: "content_editor"
                capabilities: "edit and enhance generated content"
                description: "Improves clarity, grammar, tone, and formatting."
                model: "llama-3.2-1b-preview"
                provider: "groq"
                goal: "Ensure publication-ready quality."
                depends: "content_creator"
                responds: "editor_in_chief"
                status: "Initialised"
        vector_store: "none"
    ```
