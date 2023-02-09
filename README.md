# Scrap and server deployment

## Installing dependencies

```sh
pip install -r requirements.txt
```

## Virtual environments

```sh
# Install pip env manager
sudo apt install python3-venv

# Create the env
python -m venv test_env

# Activate the env
source test_env/bin/activate

# Deactivate
deactivate
```

## Running scrapper

```sh
python src
```

## Running tests

```sh
python tests
```

## Running server

```sh
python src/__main__.py server
```

## Running frontend

```sh
cd frontend && npm start
```