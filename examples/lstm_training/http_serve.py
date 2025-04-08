import os
import json
import numpy as np
import pandas as pd
from tensorflow.keras.models import load_model
from sklearn.preprocessing import MinMaxScaler
import joblib
from flask import Flask, request, jsonify
import re  # For cleaning symbol names
import warnings  # Import the warnings module

app = Flask(__name__)

# --- Configuration (These should match your training script) ---
MODEL_SAVE_DIR = 'models'
FEATURES = ['open', 'close', 'high', 'low', 'volume']
TARGET_COLUMN = 'close'

# --- Global Variables (Loaded on Startup) ---
model_cache = {}  # Stores models, key is (symbol, window_size, forecast_size), value is model
scaler_cache = {}  # Stores scalers, key is (symbol, window_size, forecast_size), value is scaler
target_info_cache = {}  # Stores target info, key is (symbol, window_size, forecast_size), value is (target_column_index, features)

def clean_symbol_name(filename):
    """Removes .json and replaces invalid chars for filenames."""
    symbol = filename.replace('.json', '')
    return symbol

def load_model_and_scaler(symbol, window_size, forecast_size):  # ADDED forecast_size
    """Loads the pre-trained model, scaler, and target index for a given symbol, window_size and forecast (or retrieves from cache)."""
    cache_key = (symbol, window_size, forecast_size)

    if cache_key in model_cache and cache_key in scaler_cache and cache_key in target_info_cache:
        print(f"Loading model from cache for: {cache_key}")
        return model_cache[cache_key], scaler_cache[cache_key], target_info_cache[cache_key]

    # Construct filename including the window_size and forecast_size
    model_file = os.path.join(MODEL_SAVE_DIR, f'model_{symbol}_w{window_size}_f{forecast_size}.h5')
    scaler_file = os.path.join(MODEL_SAVE_DIR, f'scaler_{symbol}_w{window_size}_f{forecast_size}.pkl')
    target_index_file = os.path.join(MODEL_SAVE_DIR, f'target_index_{symbol}_w{window_size}_f{forecast_size}.json')

    try:
        print(f"Attempting to load model: {model_file}")  # Debugging
        with warnings.catch_warnings():
            warnings.simplefilter("ignore")  # Ignore all warnings within this block
            model = load_model(model_file)
        print(f"Successfully loaded model: {model_file}")  # Debugging
    except Exception as e:
        print(f"ERROR loading model {model_file}: {type(e).__name__} - {e}")
        return None, None, None

    try:
        scaler = joblib.load(scaler_file)
    except Exception as e:
        print(f"Error loading scaler {scaler_file}: {e}")
        return None, None, None

    try:
        with open(target_index_file, 'r') as f:
            target_data = json.load(f)
            target_column_index = target_data['target_column_index']
            features = target_data['features']
            target_info = (target_column_index, features)  # Tuple to store everything
    except Exception as e:
        print(f"Error loading target index {target_index_file}: {e}")
        return None, None, None

    model_cache[cache_key] = model
    scaler_cache[cache_key] = scaler
    target_info_cache[cache_key] = target_info  # Store in cache

    return model, scaler, target_info

@app.route('/predict', methods=['POST'])
def predict():
    """Endpoint for receiving OHLCV data and returning a prediction."""
    data = request.get_json()

    if not data or 'symbol' not in data or 'ohlcv' not in data or 'window_size' not in data or 'forecast_size' not in data:  # Added 'forecast_size' check
        return jsonify({'error': 'Invalid request.  Requires "symbol", "ohlcv", "window_size", and "forecast_size" in the JSON payload.'}), 400

    symbol = clean_symbol_name(data['symbol'])  # Sanitize the symbol
    ohlcv_data = data['ohlcv']
    window_size = data['window_size']  # Get window_size from request
    forecast_size = data['forecast_size']  # Get forecast_size from request #Added Forecast_size and test

    if not isinstance(window_size, int) or window_size <= 0:
        return jsonify({'error': '"window_size" must be a positive integer.'}), 400

    if not isinstance(forecast_size, int) or forecast_size <= 0: #added Forecast_size and test
        return jsonify({'error': '"forecast_size" must be a positive integer.'}), 400

    if not isinstance(ohlcv_data, list) or len(ohlcv_data) != window_size:  # Dynamic length check
        return jsonify({'error': f'OHLCV data must be a list of length {window_size}.'}), 400  # Dynamic error message

    # Load the pre-trained model and scaler (from cache)
    model, scaler, target_info = load_model_and_scaler(symbol, window_size, forecast_size)  # Pass forecast_size to loader #ADDED Forecast Size
    if model is None or scaler is None or target_info is None:
        return jsonify({'error': f'Could not load model or scaler for symbol: {symbol} with window_size: {window_size} and forecast_size: {forecast_size}'}), 500 #added forecast size

    target_column_index, features = target_info

    # --- Data Validation ---
    for bar in ohlcv_data:
        if not all(key in bar for key in FEATURES):  # Check for all required keys
            return jsonify({'error': f'Each OHLCV bar must contain the keys: {FEATURES}'}), 400

    # --- Preprocess Data ---
    try:
        df = pd.DataFrame(ohlcv_data)
        df = df[features]  # Ensure correct column order and only use the specified features
        scaled_data = scaler.transform(df)  # Use .transform, NOT .fit_transform
    except Exception as e:
        print(f"Error during data preprocessing: {e}")
        return jsonify({'error': f'Error during data preprocessing: {e}'}), 500

    # Reshape the data for LSTM input (batch_size, timesteps, features)
    reshaped_data = np.reshape(scaled_data, (1, window_size, len(features)))  # Dynamic window_size

    # --- Make Prediction ---
    try:
        prediction = model.predict(reshaped_data)[0][0]  # Get the single prediction value
    except Exception as e:
        print(f"Error during prediction: {e}")
        return jsonify({'error': f'Error during prediction: {e}'}), 500

    # --- Inverse Transform the Prediction ---
    # Create a zero-filled array with the same number of features as the scaled data
    dummy_array = np.zeros((1, len(features)))

    # Place the prediction into the correct column
    dummy_array[0, target_column_index] = prediction

    # Inverse transform the dummy array to get the unscaled prediction
    unscaled_prediction = scaler.inverse_transform(dummy_array)[0, target_column_index]

    return jsonify({'prediction': float(unscaled_prediction)})  # Ensure it's JSON serializable

if __name__ == '__main__':
    # Preload models and scalers on startup.
    print("Preloading models and scalers...")
    loaded_models = set()  # Track already loaded (symbol, window_size, forecast_size) to prevent infinite loops

    for filename in os.listdir(MODEL_SAVE_DIR):
        if filename.startswith('model_') and filename.endswith('.h5'):
            # Extract symbol, window_size, and forecast_size from filename
            match = re.match(r"model_(.+)_w(\d+)_f(\d+)\.h5", filename)  # Updated regex
            if match:
                symbol = match.group(1)
                window_size = int(match.group(2))
                forecast_size = int(match.group(3)) # NEW
                cache_key = (symbol, window_size, forecast_size)  # Key is tuple
                if cache_key not in loaded_models:
                    load_model_and_scaler(symbol, window_size, forecast_size)  # Updated call
                    loaded_models.add(cache_key)  # Store cache key.
                    print(f"Preloaded model for symbol: {symbol}, window_size: {window_size}, forecast_size: {forecast_size}")  # Updated print
                else:
                    print(f"Skipping preload for {cache_key}: Already loaded.")
            else:
                print(f"Warning: Could not extract symbol, window_size and forecast_size from filename: {filename}")

    print("Models and scalers preloaded.")
    app.run(debug=True, host='0.0.0.0', port=5000, use_reloader=False)  # Listen on all interfaces, disable reloader
