# Comics MVP: Server, Scrapper and Interface

## Scrapper and Server deployment (Python and Go)

### Virtual environment (Optional)

```sh
# Install pip env manager
sudo apt install python3-venv
## python -m pip install --upgrade pip

# Create the env
python -m venv comics_env
# Activate the env
source comics_env/bin/activate
# Deactivate
deactivate
```

> [!IMPORTANT]
> For corrupted virtual env:
> ```sh
> rm -rf comics_env
> python3 -m venv comics_env
> source comics_env/bin/activate
> ```

### Installing dependencies

```sh
pip install -r requirements.txt

# after installing new dependencies run
pip freeze > requirements.txt

## check updates
pip-review --local
## apply them
pip-review --auto
```

> [!NOTE]
> On termux there are several dependencies that need to be installed manually, and they can take hours to install.
> ```sh
> pkg install c-ares
> GRPC_PYTHON_DISABLE_LIBC_COMPATIBILITY=1 GRPC_PYTHON_BUILD_SYSTEM_OPENSSL=1 GRPC_PYTHON_BUILD_SYSTEM_ZLIB=1 GRPC_PYTHON_BUILD_SYSTEM_CARES=1 CFLAGS+=" -U__ANDROID_API__ -D__ANDROID_API__=30 -include unistd.h" LDFLAGS+=" -llog" pip install grpcio
> ```

### Running scrapper

```sh
python src
```

### Running tests

```sh
python -m unittest discover -s tests
```

```sh
env PYTHONPATH=src python3 -m pytest src/*/*_test.py -v
```

### Running server

```sh
npm run server

# Or for debug
python src/__main__.py server debug
```

### Running Go server

```sh
(cd go_server && go run ./cmd/server/main.go)
```

### Deployment on Heroku

```sh
git push heroku
heroku logs --tail # debug
```

### Deployment on Render

Should be triggered with every commit to the main branch on github repo

## Interface deployment (React JS)

### Running frontend

```sh
npm install
npm start
```

Runs the app in the development mode.\
Open [http://localhost:3000](http://localhost:3000) to view it in the browser.

The page will reload when you make changes.\
You may also see any lint errors in the console.

### Running tests

```sh
npm test
```

Launches the test runner in the interactive watch mode.\
See the section about [running tests](https://facebook.github.io/create-react-app/docs/running-tests) for more information.

### Running build and deployment

```sh
npm run build

# GH-pages
npm run deploy && git push origin
```

Builds the app for production to the `build` folder.\
It correctly bundles React in production mode and optimizes the build for the best performance.

The build is minified and the filenames include the hashes.\
Your app is ready to be deployed!

See the section about [deployment](https://facebook.github.io/create-react-app/docs/deployment) for more information.
