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
WINDOW_SIZE = 60
FEATURES = ['open', 'close', 'high', 'low', 'volume']
TARGET_COLUMN = 'close'

# --- Global Variables (Loaded on Startup) ---
model_cache = {}  # Stores models, key is symbol, value is model
scaler_cache = {}  # Stores scalers, key is symbol, value is scaler
target_info_cache = {}  # Stores target info, key is symbol, value is (target_column_index, features)


def clean_symbol_name(filename):
    """Removes .json and replaces invalid chars for filenames."""
    symbol = filename.replace('.json', '')
    #symbol = re.sub(r'[\\/*?:"<>|]', '_', symbol)  # Replace invalid chars <--- Remove
    return symbol

def load_model_and_scaler(symbol):
    """Loads the model, scaler, and target index for a given symbol (or retrieves from cache)."""

    if symbol in model_cache and symbol in scaler_cache and symbol in target_info_cache:
        return model_cache[symbol], scaler_cache[symbol], target_info_cache[symbol]

    model_file = os.path.join(MODEL_SAVE_DIR, f'model_{symbol}.h5')
    scaler_file = os.path.join(MODEL_SAVE_DIR, f'scaler_{symbol}.pkl')
    target_index_file = os.path.join(MODEL_SAVE_DIR, f'target_index_{symbol}.json')

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

    model_cache[symbol] = model
    scaler_cache[symbol] = scaler
    target_info_cache[symbol] = target_info  # Store in cache

    return model, scaler, target_info

@app.route('/predict', methods=['POST'])
def predict():
    """Endpoint for receiving OHLCV data and returning a prediction."""
    data = request.get_json()

    if not data or 'symbol' not in data or 'ohlcv' not in data:
        return jsonify({'error': 'Invalid request.  Requires "symbol" and "ohlcv" in the JSON payload.'}), 400

    symbol = clean_symbol_name(data['symbol'])  # Sanitize the symbol
    ohlcv_data = data['ohlcv']

    if not isinstance(ohlcv_data, list) or len(ohlcv_data) != WINDOW_SIZE:
        return jsonify({'error': f'OHLCV data must be a list of length {WINDOW_SIZE}.'}), 400

    # Load the model and scaler (from cache)
    model, scaler, target_info = load_model_and_scaler(symbol)
    if model is None or scaler is None or target_info is None:
        return jsonify({'error': f'Could not load model or scaler for symbol: {symbol}'}), 500

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
    reshaped_data = np.reshape(scaled_data, (1, WINDOW_SIZE, len(features)))

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
    loaded_symbols = set()  # Track already loaded symbols to prevent infinite loops

    for filename in os.listdir(MODEL_SAVE_DIR):
        if filename.startswith('model_') and filename.endswith('.h5'):
            # Extract symbol from filename *correctly*
            match = re.match(r"model_(.+)\.h5", filename)
            if match:
                symbol = match.group(1)
                print(f"Extracted symbol: {symbol} from filename: {filename}")  # Debug print
                # Prevent Infinite Loops: Check if already loaded BEFORE calling load_model_and_scaler()
                if symbol not in loaded_symbols:
                    load_model_and_scaler(symbol)
                    loaded_symbols.add(symbol)
                else:
                    print(f"Warning: Model for symbol '{symbol}' already loaded. Skipping.")
            else:
                print(f"Warning: Could not extract symbol from filename: {filename}")

    print("Models and scalers preloaded.")
    app.run(debug=True, host='0.0.0.0', port=5000, use_reloader=False)  # Listen on all interfaces, disable reloader
