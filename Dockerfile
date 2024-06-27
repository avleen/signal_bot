FROM python:3.12-alpine

# Set the working directory
WORKDIR /app

# Install any needed packages specified in requirements.txt
RUN apk add --update --no-cache --virtual .tmp-build-deps gcc libc-dev g++ rust cargo geos-dev
COPY requirements.txt /app/python/requirements.txt
RUN pip install --prefer-binary -r python/requirements.txt
RUN apk del .tmp-build-deps

# Copy the current directory contents into the container at /app
COPY . /app
COPY auth.json /app/python/auth.json

# Set the environment here, or add this to your docker-compose.yml
# ENV IMAGEDIR=/path/to/tmp/images/dir
# ENV STATEDB=/path/to/messages.db
# ENV URL=signal:8889
# ENV REST_URL=signal_native:8888
# ENV PHONE=+1234567890
# ENV HOURS=24
# ENV GOOGLE_APPLICATION_CREDENTIALS=/path/to/auth.json

# Start the bot
WORKDIR /app/python
CMD ["python", "wsclient.py"]
