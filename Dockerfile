# set base image (host OS)
FROM python:3.10.12 AS builder

# copy the dependencies file to the working directory
COPY requirements.txt .

# install dependencies
RUN pip install --user -r requirements.txt

# second unnamed stage
FROM python:3.10.12-slim
# set the working directory in the container
WORKDIR /code

# copy only the dependencies installation from the 1st stage image
COPY --from=builder /root/.local /root/.local
# copy the content of the local src directory to the working directory
COPY src .

# update PATH environment variable
ENV PATH=/root/.local:$PATH

# command to run on container start
# CMD [ "ls" ]
CMD [ "python", ".", "server" ]