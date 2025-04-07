import os
import json
import re
import numpy as np
import tensorflow as tf
from flask import Flask, request, jsonify
import joblib # To load the scaler
import logging

# --- Configuration ---
MODEL_SAVE_DIR = 'models' # Directory where models and scalers are saved
# This MUST match the WINDOW_SIZE used during training
EXPECTED_WINDOW_SIZE = 20

# --- Flask App Initialization ---
app = Flask(__name__)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# --- Helper Function ---
def clean_symbol_name(filename_or_symbol):
    """Removes .json and replaces invalid chars for filenames/paths."""
    symbol = filename_or_symbol.replace('.json', '')
    symbol = re.sub(r'[\\/*?:"<>|]', '_', symbol) # Replace invalid chars
    return symbol

# --- Prediction Endpoint ---
@app.route('/predict/<symbol>', methods=['POST'])
def predict(symbol):
    logger.info(f"Received prediction request for symbol: {symbol}")

    # 1. Clean symbol name and construct file paths
    cleaned_symbol = clean_symbol_name(symbol)
    model_file = os.path.join(MODEL_SAVE_DIR, f'model_{cleaned_symbol}.h5')
    scaler_file = os.path.join(MODEL_SAVE_DIR, f'scaler_{cleaned_symbol}.pkl')
    target_index_file = os.path.join(MODEL_SAVE_DIR, f'target_index_{cleaned_symbol}.json')

    # 2. Check if required files exist
    if not os.path.exists(model_file):
        logger.error(f"Model file not found: {model_file}")
        return jsonify({'error': f'Model for symbol {symbol} not found'}), 404
    if not os.path.exists(scaler_file):
        logger.error(f"Scaler file not found: {scaler_file}")
        return jsonify({'error': f'Scaler for symbol {symbol} not found'}), 404
    if not os.path.exists(target_index_file):
        logger.error(f"Target index file not found: {target_index_file}")
        return jsonify({'error': f'Target index info for symbol {symbol} not found'}), 404

    try:
        # 3. Load Model, Scaler, and Target Index Info
        logger.info(f"Loading artifacts for {cleaned_symbol}...")
        model = tf.keras.models.load_model(model_file)
        scaler = joblib.load(scaler_file)
        with open(target_index_file, 'r') as f:
            target_info = json.load(f)
        target_col_index = target_info['target_column_index']
        features_list = target_info['features'] # List of features in the order they were trained
        num_features = len(features_list)
        logger.info(f"Artifacts loaded. Expected features ({num_features}): {features_list}")
        logger.info(f"Target column index to predict: {target_col_index}")

        # 4. Get and Validate Input Data
        data = request.get_json()
        if not data:
            logger.error("No JSON data received in request.")
            return jsonify({'error': 'No input data received'}), 400

        # Convert to numpy array
        # Expecting a list of lists/arrays, shape (WINDOW_SIZE, num_features)
        raw_features = np.array(data, dtype=np.float64)

        # --- INPUT VALIDATION ---
        if raw_features.ndim != 2:
             logger.error(f"Input data has incorrect dimensions: {raw_features.ndim} (expected 2)")
             return jsonify({'error': f'Input data must be a 2D array/list of lists.'}), 400

        if raw_features.shape[0] != EXPECTED_WINDOW_SIZE:
            logger.error(f"Input data has incorrect number of time steps: {raw_features.shape[0]} (expected {EXPECTED_WINDOW_SIZE})")
            return jsonify({'error': f'Input data must contain exactly {EXPECTED_WINDOW_SIZE} time steps'}), 400

        if raw_features.shape[1] != num_features:
            logger.error(f"Input data has incorrect number of features: {raw_features.shape[1]} (expected {num_features})")
            logger.error(f"Expected Features: {features_list}")
            # Log received data structure *carefully* if needed for debugging, maybe just the first row shape
            # logger.debug(f"First row of received data (structure check): {data[0] if data else 'None'}")
            return jsonify({'error': f'Input data must contain {num_features} features per time step. Check order.'}), 400
        # --- END INPUT VALIDATION ---

        logger.info(f"Received input data shape: {raw_features.shape}")

        # 5. Preprocess (Scale) the Input Data
        # Scaler expects shape (n_samples, n_features)
        logger.info("Scaling input features...")
        scaled_features = scaler.transform(raw_features)
        logger.debug(f"Sample scaled features (first row): {scaled_features[0]}")


        # 6. Reshape for LSTM Input
        # LSTM expects (batch_size, timesteps, features)
        lstm_input = scaled_features.reshape((1, EXPECTED_WINDOW_SIZE, num_features))
        logger.info(f"Reshaped input for LSTM: {lstm_input.shape}")

        # 7. Make Prediction
        logger.info("Making prediction...")
        scaled_prediction = model.predict(lstm_input) # Output shape is likely (1, 1)
        logger.info(f"Raw scaled prediction: {scaled_prediction}")

        # 8. Postprocess (Inverse Scale) the Prediction
        logger.info("Inverse scaling prediction...")
        # Create a dummy array with the same number of features as the scaler expects
        # We need to place the scaled prediction into the correct column index
        dummy_inverse_input = np.zeros((1, num_features))
        # Place the single prediction value into the target column
        dummy_inverse_input[0, target_col_index] = scaled_prediction[0, 0]

        # Inverse transform using the loaded scaler
        inversed_output = scaler.inverse_transform(dummy_inverse_input)

        # Extract the inverse-scaled value from the target column
        final_prediction = inversed_output[0, target_col_index]
        logger.info(f"Final inverse-scaled prediction: {final_prediction}")

        # 9. Return Prediction
        return jsonify({'prediction': final_prediction})

    except FileNotFoundError: # Should be caught above, but as fallback
         logger.exception(f"File not found error during processing for symbol {symbol}")
         return jsonify({'error': f'Artifacts for symbol {symbol} not found'}), 404
    except ValueError as ve:
        logger.exception(f"ValueError during processing for symbol {symbol}: {ve}")
        # Common causes: shape mismatch during scaling/prediction, non-numeric data
        return jsonify({'error': f'Data processing error: {ve}'}), 400
    except Exception as e:
        logger.exception(f"An unexpected error occurred for symbol {symbol}: {e}")
        return jsonify({'error': f'An unexpected error occurred: {str(e)}'}), 500

# --- Run Flask App ---
if __name__ == '__main__':
    # Use host='0.0.0.0' to make it accessible on your network
    # debug=True automatically reloads on code changes, but disable for production
    app.run(debug=True, host='0.0.0.0', port=5000)
