# Signal Bot

A bot for your Signal groups. It was originally designed to provide summaries
of busy chats through an LLM (ChatGPT / Gemini).

The current maintained bot is writted in Go and can be found in the `go/` directory.

# Commands

1. `!summary <hours | num_messages>`: Generate a summary of the chat.
Default: last 24 hours.
1. `!ask <question>`: Ask a question based on the chat history.
Example: `!ask what links were posted today?`
1. `!imagine <prompt>`: Generate an image.

# Setup

The bot is set up to run in a container. This isn't required but it'll make your life easier.

## Dependencies

1. A Signal account. You can use your own, or create a dedicated account with a new phone number.
1. A place to run the bot and dependencies.
1. [Signal CLI Rest API](https://github.com/bbernhard/signal-cli-rest-api) running in it's own container.
1. If you want to use OpenAI for LLM actions, you'll need to get an API key.
If you want to use Google Gemini / Vertex for LLM actions, you'll need to set up a Google Cloud account, set up an account, enable the Vertex API.

Currently the bot is assume you'll be using BOTH services and expects to find keys / credentials for them.
This will become more flexible in the future.

## Caveats
For performance reasons we run `signal-cli-rest-api` in `json-rpc` mode. This receives messages sent to the Signal account in real time. If your bot instance is not running, any incoming messages are lost.

`signal-cli-rest-api` does have another mode where messages can be fetched on demand, but it's slightly slower. Support for it would be easy to add and patches are welcome.

One of the key features of using Signal is the end-to-end security guarantees.
Because this bot listens to and saves messages on disk unencrypted, the guarantee is broken.
The bot does have a `MAX_AGE` setting for how many hours messages are stored but you should ensure any participants in chats are comfortable with this behavior.

## Installation

1. Follow the instructions on the Create the [Signal CLI Rest API](https://github.com/bbernhard/signal-cli-rest-api) page to install and configure the app.
A docker-compose config like this may be helpful:
    ```
    signal:
        image: bbernhard/signal-cli-rest-api:latest
        container_name: signal
        environment:
            MODE: 'native'
            PORT: 9999
        volumes:
            - '/var/lib/docker/signal-cli:/home/.local/share/signal-cli'
        ports:
            - "9999:9999"
        restart: unless-stopped
        logging:
            driver: json-file
    ```
1. Start the REST API:
    ```
    docker-compose up -d signal
    ```
1. Once the REST API container is running, connect it to the Signal account you'll be using.
1. Update the configuration and change `MODE: 'native'` to `MODE: 'json-rpc'`. Restart `signal-cli-rest-api`.
1. Clone the `signal_bot` repo:
    ```
    git clone https://github.com/avleen/signal_bot.git
    cd signal_bot
    ```
1. You need to create two files:
    * `go/auth.json` which containers your gcloud credentials (follow the Google Cloud auth instructions for generating this).
    * `common/prompt_summary.txt` which contains the summary request prompt. Your chat history will be appended to this before it's sent to OpenAI / Google.
1. Build the container:
    ```
    docker build . -t signal_bot:latest
    ```
1. Set up your docker compose file:
    ```
    signal_bot:
        image: signal_bot:latest
        tty: true
        depends_on:
            signal:
                condition: service_started
        restart: unless-stopped
        environment:
            - GOOGLE_APPLICATION_CREDENTIALS=/app/auth.json
            - GOOGLE_TEXT_MODEL=gemini-1.5-flash-001
            - IMAGE_PROVIDER=openai
            - IMAGEDIR=/var/lib/signal/images
            - LOCATION=us-central1
            - MAX_AGE=24
            - OPENAI_API_KEY=<openapi_key>
            - PHONE=+123456789
            - PROJECT_ID=<google_cloud_project_id>
            - REST_URL=signal:9999
            - STATEDB=/var/lib/signal/messages.db
            - SUMMARY_PROVIDER=google
            - URL=signal:9999
        volumes:
            - /var/lib/docker/signal-cli/state:/var/lib/signal
        logging:
            driver: json-file
    ```
1. Start the bot:
    ```
    docker-compose up -d signal_bot
    docker-compose logs --tail 50 -t -f signal_bot
    ```
1. In your chat try to generate a summary. Errors are printed in the container log:
    ```
    !summary
    ```