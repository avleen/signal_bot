#!/usr/bin/env python

import argparse
import base64
import os
import time
from openai import OpenAI

"""Generates an image from a text prompt using the Image Generation model."""

def init_outputdir(outputdir):
    # Create the output directory if it doesn't exist
    try:
        os.makedirs(outputdir)
    except FileExistsError:
        pass

def imagine(prompt, outputdir, requestor="None"):
    # Create the output directory if it doesn't exist
    init_outputdir(outputdir)

    # Initialize the OpenAI client
    client = OpenAI()

    # Generate an image from the prompt
    try:
        images = client.images.generate(
            model="dall-e-3",
            prompt=prompt,
            size="1024x1024",
            quality="standard",
            n=1,
            response_format="b64_json"
        )
        revised_prompt = images.data[0].revised_prompt
        data = images.data[0].b64_json
        # Decode the base64 data
        data = base64.b64decode(data)
        filename = f"{outputdir}/{time.time()}-{requestor}.png"
        with open(filename, "wb") as f:
            f.write(data)
        print(f"INFO: Image saved to {filename}, revised prompt: {revised_prompt}")
        return filename, revised_prompt
    except Exception as e:
        print(f"ERROR: {str(e)}")
        # Re-raise the exception so the caller can handle it
        raise e
    
if __name__ == "__main__":
    # Command line arguments:
    # --prompt: The text prompt to generate an image from
    # --output: The output file to save the image to

    parser = argparse.ArgumentParser(description="Generate an image from a text prompt")
    parser.add_argument("--prompt", help="The text prompt to generate an image from", required=True)
    parser.add_argument("--outputdir", help="The output directory to save the image to", required=True)
    args = parser.parse_args()
    imagine(args.prompt, args.outputdir)
