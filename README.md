# Comics MVP: Server, Scrapper and Interface

## Scrapper and Server deployment (Python)

### Virtual environment (Optional)

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

### Installing dependencies

```sh
pip install -r requirements.txt
```

### Running scrapper

```sh
python src
```

### Running tests

```sh
python tests
```

### Running server

```sh
python src/__main__.py server
```
## Interface deployment (React)

### Running frontend

```sh
npm install --prefix frontend
npm start --prefix frontend
```