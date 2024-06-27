#!/usr/bin/env python

import argparse
import os
import time
import vertexai
from google.api_core import exceptions
from vertexai.preview.vision_models import ImageGenerationModel

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

    # Initialize the Vertex AI client
    project_id = "tmp-k8s-tutorial"
    location = "us-central1"
    vertexai.init(project=project_id, location=location)
    model = ImageGenerationModel.from_pretrained("imagegeneration@006")

    # Generate an image from the prompt
    try:
        images = model.generate_images(
            prompt=prompt,
            # Optional parameters
            number_of_images=1,
            language="en",
            # You can't use a seed value and watermark at the same time.
            # add_watermark=False,
            # seed=100,
            aspect_ratio="1:1",
            safety_filter_level="block_few",
            person_generation="allow_adult",
            
        )
    except exceptions.InvalidArgument as e:
        print(f"ERROR: {str(e)}")
        # Re-raise the exception so the caller can handle it
        raise e
    
    try:
        # Generate a filename to save the image to, in the format:
        # <outputdir>/<requestor>-<timestamp>.png
        # For the timestamp we'll use YYYYDDMM-HHMMSS
        timestamp = time.strftime("%Y%m%d-%H%M%S")
        filename = f"{outputdir}/{requestor}-{timestamp}.png"
        images[0].save(location=filename, include_generation_parameters=False)
        print(f"INFO: Image saved to {filename}")
        return filename
    except IndexError:
        print(f"ERROR: No images generated: Google probably didn't like the prompt '{prompt}'.")
        raise ValueError(f"No images generated: Google probably didn't like the prompt '{prompt}'.")

if __name__ == "__main__":
    # Command line arguments:
    # --prompt: The text prompt to generate an image from
    # --output: The output file to save the image to

    parser = argparse.ArgumentParser(description="Generate an image from a text prompt")
    parser.add_argument("--prompt", help="The text prompt to generate an image from", required=True)
    parser.add_argument("--outputdir", help="The output directory to save the image to", required=True)
    args = parser.parse_args()
    imagine(args.prompt, args.outputdir)
