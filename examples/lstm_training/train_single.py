import os
import json
import argparse
import re
import numpy as np
import pandas as pd
import joblib
import logging
from sklearn.preprocessing import MinMaxScaler
from tensorflow.keras.models import Sequential, load_model
from tensorflow.keras.layers import LSTM, Dense, Dropout
from tensorflow.keras.optimizers import Adam
from tensorflow.keras.callbacks import EarlyStopping, ReduceLROnPlateau # Added Callbacks

# --- Configuration (Keep consistent with server.py and train.py) ---
MODEL_SAVE_DIR = 'models' # Directory to save models and scalers
WINDOW_SIZE = 20          # Number of past bars to use for prediction
FORECAST_SIZE = 1         # Number of steps ahead to predict
FEATURES = ['open', 'close', 'high', 'low', 'volume', 'body', 'hl', 'wickUp', 'wickDn', 'sma8', 'sma20', 'sma200']
TARGET_COLUMN = 'close'   # The column we want to predict
EPOCHS = 50               # Consider making this an argument or adjusting as needed
BATCH_SIZE = 32
VALIDATION_SPLIT = 0.1    # Split for validation during training
MIN_DATA_POINTS = WINDOW_SIZE + FORECAST_SIZE + 20 # Minimum data points needed

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

# --- Helper Functions (Mostly identical to train.py) ---

def clean_symbol_name(filename):
    """Removes .json and replaces invalid chars for filenames."""
    symbol = filename.replace('.json', '')
    # Replace common separators and any other invalid filesystem characters
    symbol = re.sub(r'[\\/*?:"<>|/-]', '_', symbol)
    return symbol

def load_data_for_symbol(filepath):
    """Loads data from a single JSON file, sorts by time, handles NaNs."""
    logging.info(f"Attempting to load data from: {filepath}")
    if not os.path.exists(filepath):
        logging.error(f"File not found: {filepath}")
        return None
    try:
        with open(filepath, 'r') as f:
            data = json.load(f)
            if not data:
                logging.warning(f"File is empty: {filepath}")
                return None
            df = pd.DataFrame(data)
            if 'time' not in df.columns:
                logging.error(f"'time' column missing in {filepath}.")
                return None

            # Ensure time is numeric and sort
            df['time'] = pd.to_numeric(df['time'])
            df = df.sort_values(by='time').reset_index(drop=True)

            # Handle potential NaN values (e.g., initial SMAs)
            # IMPORTANT: Check if this fill strategy makes sense for your data
            df = df.fillna(method='bfill').fillna(method='ffill') # Backfill then forward fill
            df = df.dropna() # Drop any remaining NaNs if ffill/bfill didn't cover edges

            if len(df) < MIN_DATA_POINTS:
                logging.warning(f"Insufficient data points ({len(df)} < {MIN_DATA_POINTS}) after loading and cleaning {filepath}.")
                return None

            logging.info(f"Successfully loaded and cleaned {len(df)} data points.")
            return df

    except json.JSONDecodeError:
        logging.error(f"Could not decode JSON from {filepath}.")
        return None
    except Exception as e:
        logging.error(f"Error loading or processing {filepath}: {e}")
        return None

def preprocess_data(df, feature_columns, target_column, window_size, forecast_size):
    """Scales data and creates sequences for a single symbol."""
    logging.info("Starting preprocessing...")
    if df is None or df.empty:
        logging.error("Preprocessing received empty DataFrame.")
        return None, None, None, None

    # Ensure correct feature columns exist
    missing_cols = [col for col in feature_columns if col not in df.columns]
    if missing_cols:
        logging.error(f"Missing required feature columns: {missing_cols}")
        return None, None, None, None

    try:
        df_features = df[feature_columns].copy()
        target_col_index = feature_columns.index(target_column)
    except KeyError as e:
        logging.error(f"Error accessing feature columns: {e}")
        return None, None, None, None
    except ValueError as e:
         logging.error(f"Target column '{target_column}' not found in features list: {e}")
         return None, None, None, None

    # Scale the features
    # IMPORTANT: A new scaler is fitted every time. If continuing training,
    # ideally you'd load the *previous* scaler. However, refitting usually
    # works okay if the data distribution hasn't changed drastically.
    # For perfect continuity, saving/loading the scaler BEFORE fitting is needed,
    # but joblib/sklearn doesn't easily allow 'updating' a fit.
    scaler = MinMaxScaler(feature_range=(0, 1))
    scaled_data = scaler.fit_transform(df_features)
    logging.info(f"Data scaled. Shape: {scaled_data.shape}")

    X, y = [], []
    # Iterate up to the point where a forecast is possible
    # Ensure we don't go out of bounds for y
    for i in range(window_size, len(scaled_data) - forecast_size + 1):
        X.append(scaled_data[i - window_size:i, :])
        y.append(scaled_data[i + forecast_size - 1, target_col_index])

    if not X or not y:
        logging.warning("No sequences created after preprocessing. Check data length and window/forecast sizes.")
        return None, None, None, None

    X = np.array(X)
    y = np.array(y)
    logging.info(f"Preprocessing complete. X shape: {X.shape}, y shape: {y.shape}")
    return X, y, scaler, target_col_index

def build_model(input_shape):
    """Builds and compiles the LSTM model."""
    model = Sequential()
    model.add(LSTM(128, return_sequences=True, input_shape=input_shape))
    model.add(Dropout(0.2))
    model.add(LSTM(64, return_sequences=False))
    model.add(Dropout(0.2))
    model.add(Dense(32, activation='relu'))
    model.add(Dense(1))
    # Consider adjusting learning rate
    model.compile(optimizer=Adam(learning_rate=0.0005), loss='mean_squared_error')
    logging.info("New model built and compiled.")
    model.summary() # Print model summary
    return model

# --- Main Training Function ---
def train_single_file(filepath):
    """Trains or continues training a model for the data in the specified file."""
    logging.info(f"--- Starting Training for: {filepath} ---")

    # 1. Prepare Symbol and File Paths
    if not os.path.isabs(filepath): # If relative path, assume it's in current dir or adjust as needed
        filepath = os.path.abspath(filepath)

    base_filename = os.path.basename(filepath)
    symbol = clean_symbol_name(base_filename)
    logging.info(f"Cleaned symbol name: {symbol}")

    # Create save directory if it doesn't exist
    os.makedirs(MODEL_SAVE_DIR, exist_ok=True)

    model_file = os.path.join(MODEL_SAVE_DIR, f'model_{symbol}.h5')
    scaler_file = os.path.join(MODEL_SAVE_DIR, f'scaler_{symbol}.pkl')
    target_index_file = os.path.join(MODEL_SAVE_DIR, f'target_index_{symbol}.json')

    # 2. Load Data
    df = load_data_for_symbol(filepath)
    if df is None:
        logging.error("Failed to load data. Aborting training.")
        return

    # 3. Preprocess Data (This fits a new scaler each time)
    X, y, scaler, target_col_index = preprocess_data(
        df, FEATURES, TARGET_COLUMN, WINDOW_SIZE, FORECAST_SIZE
    )
    if X is None or y is None or scaler is None:
        logging.error("Failed to preprocess data. Aborting training.")
        return

    # 4. Build or Load Model
    model = None
    if os.path.exists(model_file):
        logging.info(f"Found existing model file: {model_file}. Loading...")
        try:
            model = load_model(model_file)
            logging.info("Existing model loaded successfully.")
            # Optional: Print summary to confirm loaded structure
            # model.summary()
        except Exception as e:
            logging.error(f"Error loading model {model_file}: {e}. Building a new one.")
            # Fall through to build new model
            model = None # Ensure we build below

    if model is None: # If model didn't exist or failed to load
        logging.info("Building new model...")
        # Ensure X has the right shape for input_shape
        if X.ndim != 3 or X.shape[1] != WINDOW_SIZE or X.shape[2] != len(FEATURES):
             logging.error(f"Unexpected shape for X: {X.shape}. Cannot determine model input shape.")
             return
        model = build_model((X.shape[1], X.shape[2])) # input_shape=(timesteps, features)

    # 5. Train the Model (callbacks added for robustness)
    logging.info(f"Starting model training for {EPOCHS} epochs...")

    # Stop training if validation loss doesn't improve for 'patience' epochs
    early_stopping = EarlyStopping(monitor='val_loss', patience=10, restore_best_weights=True)
    # Reduce learning rate if validation loss plateaus
    reduce_lr = ReduceLROnPlateau(monitor='val_loss', factor=0.2, patience=5, min_lr=1e-6)

    history = model.fit(
        X, y,
        epochs=EPOCHS,
        batch_size=BATCH_SIZE,
        validation_split=VALIDATION_SPLIT,
        verbose=1,
        shuffle=True,
        callbacks=[early_stopping, reduce_lr] # Add callbacks
    )

    logging.info("Model training finished.")

    # 6. Save Artifacts
    logging.info(f"Saving artifacts for symbol: {symbol}")
    try:
        model.save(model_file)
        logging.info(f"Model saved to: {model_file}")

        joblib.dump(scaler, scaler_file)
        logging.info(f"Scaler saved to: {scaler_file}")

        target_info = {'target_column_index': target_col_index, 'features': FEATURES}
        with open(target_index_file, 'w') as f:
            json.dump(target_info, f, indent=2) # Added indent for readability
        logging.info(f"Target index info saved to: {target_index_file}")

    except Exception as e:
        logging.error(f"Error saving artifacts for {symbol}: {e}")

    logging.info(f"--- Training complete for: {symbol} ---")

# --- Script Execution ---
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Train or continue training an LSTM model for a single stock data JSON file.")
    parser.add_argument("filepath", help="Path to the input JSON file containing stock data.")
    args = parser.parse_args()

    train_single_file(args.filepath)
