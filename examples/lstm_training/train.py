import os
import json
import numpy as np
import pandas as pd
from tensorflow.keras.models import Sequential
from tensorflow.keras.layers import LSTM, Dense, Dropout
from tensorflow.keras.optimizers import Adam
from tensorflow.keras.models import load_model
from sklearn.preprocessing import MinMaxScaler
import joblib # For saving/loading the scaler
import re # For cleaning symbol names

# --- Configuration ---
DATA_DIRECTORY = 'data'
MODEL_SAVE_DIR = 'models' # Directory to save models and scalers
WINDOW_SIZE = 20  # Number of past bars to use for prediction
FORECAST_SIZE = 1   # Number of steps ahead to predict (let's start with 1)
FEATURES = ['open', 'close', 'high', 'low', 'volume', 'body', 'hl', 'wickUp', 'wickDn', 'sma8', 'sma20', 'sma200']
TARGET_COLUMN = 'close' # The column we want to predict
EPOCHS = 50
BATCH_SIZE = 32
VALIDATION_SPLIT = 0.1 # Use less for validation if data per symbol is limited
MIN_DATA_POINTS = WINDOW_SIZE + FORECAST_SIZE + 20 # Minimum data points needed per file to train

# Create save directory if it doesn't exist
os.makedirs(MODEL_SAVE_DIR, exist_ok=True)

# --- Helper Functions ---

def clean_symbol_name(filename):
    """Removes .json and replaces invalid chars for filenames."""
    symbol = filename.replace('.json', '')
    symbol = re.sub(r'[\\/*?:"<>|]', '_', symbol) # Replace invalid chars
    return symbol

def load_data_for_symbol(filepath):
    """Loads data from a single JSON file and sorts by time."""
    try:
        with open(filepath, 'r') as f:
            data = json.load(f)
            if not data: # Check if file is empty
                print(f"Warning: File {filepath} is empty.")
                return None
            df = pd.DataFrame(data)
            if 'time' not in df.columns:
                print(f"Warning: 'time' column missing in {filepath}. Skipping.")
                return None
            # Ensure time is numeric (if not already) and sort
            df['time'] = pd.to_numeric(df['time'])
            df = df.sort_values(by='time').reset_index(drop=True)
            # Fill potential NaN values (e.g., initial SMAs)
            df = df.fillna(method='bfill').fillna(method='ffill') # Backfill then forward fill
            df = df.dropna() # Drop any remaining NaNs if ffill/bfill didn't cover edges
            return df
    except json.JSONDecodeError:
        print(f"Error: Could not decode JSON from {filepath}. Skipping.")
        return None
    except Exception as e:
        print(f"Error loading {filepath}: {e}")
        return None

def preprocess_data(df, feature_columns, target_column, window_size, forecast_size):
    """Scales data and creates sequences for a single symbol."""
    if df is None or df.empty:
        return None, None, None, None

    if len(df) < MIN_DATA_POINTS:
        print(f"Warning: Insufficient data points ({len(df)} < {MIN_DATA_POINTS}). Skipping preprocessing.")
        return None, None, None, None

    # Ensure correct column order
    try:
        df_features = df[feature_columns].copy()
    except KeyError as e:
        print(f"Error: Missing expected feature column: {e}. Skipping preprocessing.")
        return None, None, None, None

    target_col_index = feature_columns.index(target_column)

    # Scale the features
    scaler = MinMaxScaler(feature_range=(0, 1))
    scaled_data = scaler.fit_transform(df_features)

    X, y = [], []
    # Iterate up to the point where a forecast is possible
    for i in range(window_size, len(scaled_data) - forecast_size + 1):
        X.append(scaled_data[i - window_size:i, :])  # Input sequence (all features)
        # Target: The value of the target column 'forecast_size' steps ahead
        y.append(scaled_data[i + forecast_size - 1, target_col_index])

    if not X or not y:
        print("Warning: No sequences created after preprocessing (likely due to data length).")
        return None, None, None, None

    return np.array(X), np.array(y), scaler, target_col_index

def build_model(input_shape):
    """Builds and compiles the LSTM model."""
    model = Sequential()
    # Increased units slightly, consider tuning
    model.add(LSTM(128, return_sequences=True, input_shape=input_shape))
    model.add(Dropout(0.2))
    model.add(LSTM(64, return_sequences=False)) # Second LSTM layer
    model.add(Dropout(0.2))
    model.add(Dense(32, activation='relu')) # Intermediate dense layer
    model.add(Dense(1)) # Output layer for predicting one value
    # Consider a slightly lower learning rate
    model.compile(optimizer=Adam(learning_rate=0.0005), loss='mean_squared_error')
    return model

# --- Main Training Loop ---
def train():
    print("Starting training process...")
    processed_files = 0
    for filename in os.listdir(DATA_DIRECTORY):
        if filename.endswith('.json'):
            filepath = os.path.join(DATA_DIRECTORY, filename)
            symbol = clean_symbol_name(filename)
            print(f"\n--- Processing Symbol: {symbol} ---")

            # Define model and scaler paths for this symbol
            model_file = os.path.join(MODEL_SAVE_DIR, f'model_{symbol}.h5')
            scaler_file = os.path.join(MODEL_SAVE_DIR, f'scaler_{symbol}.pkl')
            target_index_file = os.path.join(MODEL_SAVE_DIR, f'target_index_{symbol}.json')

            # Load data for the current symbol
            df = load_data_for_symbol(filepath)
            if df is None:
                continue

            # Preprocess data
            print(f"Preprocessing data for {symbol}...")
            X, y, scaler, target_col_index = preprocess_data(
                df, FEATURES, TARGET_COLUMN, WINDOW_SIZE, FORECAST_SIZE
            )

            if X is None or y is None or scaler is None:
                print(f"Skipping training for {symbol} due to preprocessing issues.")
                continue

            print(f"Data shapes - X: {X.shape}, y: {y.shape}")

            # Build or load model
            if os.path.exists(model_file):
                print(f"Loading existing model: {model_file}")
                try:
                    model = load_model(model_file)
                except Exception as e:
                    print(f"Error loading model {model_file}: {e}. Building a new one.")
                    model = build_model((X.shape[1], X.shape[2]))
            else:
                print("Building new model...")
                model = build_model((X.shape[1], X.shape[2]))

            model.summary() # Print model summary

            # Train the model
            print(f"Training model for {symbol}...")
            history = model.fit(
                X, y,
                epochs=EPOCHS,
                batch_size=BATCH_SIZE,
                validation_split=VALIDATION_SPLIT,
                verbose=1,
                shuffle=True # Shuffle data for better training
            )

            # Save the model, scaler, and target index
            print(f"Saving model, scaler, and target index for {symbol}...")
            model.save(model_file)
            joblib.dump(scaler, scaler_file)
            with open(target_index_file, 'w') as f:
                json.dump({'target_column_index': target_col_index, 'features': FEATURES}, f)

            print(f"Successfully processed and saved artifacts for {symbol}.")
            processed_files += 1

    print(f"\n--- Training complete. Processed {processed_files} files. ---")
    if processed_files == 0:
        print("Warning: No files were successfully processed. Check data and configuration.")

if __name__ == "__main__":
    train()
