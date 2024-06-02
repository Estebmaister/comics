# Comics MVP: Server, Scrapper and Interface

## Scrapper and Server deployment (Python)

### Virtual environment (Optional)

```sh
# Install pip env manager
sudo apt install python3-venv

# Create the env
python -m venv comics_env

# Activate the env
source comics_env/bin/activate

# Deactivate
deactivate
```

### Installing dependencies

```sh
pip install -r requirements.txt

# after installing new dependencies run
pip freeze > requirements.txt
```

### Running scrapper

```sh
python src
```

### Running tests

```sh
python test
```

### Running server

```sh
npm run server

# Or for debug
python src/__main__.py server debug
```

### Deployment on Heroku

```sh
git push heroku
heroku logs --tail # debug
```
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
