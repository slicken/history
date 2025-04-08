import os
import json
import numpy as np
import pandas as pd
from tensorflow.keras.models import Sequential
from tensorflow.keras.layers import LSTM, Dense, Dropout
from tensorflow.keras.optimizers import Adam
from tensorflow.keras.models import load_model
from sklearn.preprocessing import MinMaxScaler
import joblib  # For saving/loading the scaler
import re  # For cleaning symbol names
import argparse  # For command-line argument parsing
from tensorflow.keras.callbacks import EarlyStopping #Import EarlyStopping

# --- Configuration ---
DATA_DIRECTORY = 'data'  # Default Data Directory
MODEL_SAVE_DIR = 'models'  # Directory to save models and scalers
#WINDOW_SIZE = 60  # Tune this!
#FORECAST_SIZE = 5  # Tune this!
FEATURES = ['open', 'close', 'high', 'low', 'volume']  # Only use OHLCV
TARGET_COLUMN = 'close'  # The column we want to predict
EPOCHS = 75  # Tune this! Increased to 75
BATCH_SIZE = 32  # Tune this! Decreased to 32
VALIDATION_SPLIT = 0.2  # Tune this! Increased to 0.2
#MIN_DATA_POINTS = WINDOW_SIZE + FORECAST_SIZE + 10 # Reduced min data points needed
TEST_SIZE = 0.1  # Percentage of the data to use for testing

# Create save directory if it doesn't exist
os.makedirs(MODEL_SAVE_DIR, exist_ok=True)

# --- Helper Functions ---

def clean_symbol_name(filename):
    """Removes .json and replaces invalid chars for filenames."""
    symbol = filename.replace('.json', '')
    #symbol = re.sub(r'[\\/*?:"<>|]', '_', symbol)  # Remove replace invalid chars <------
    return symbol

def load_data_for_symbol(filepath):
    """Loads data from a single JSON file and sorts by time."""
    try:
        with open(filepath, 'r') as f:
            data = json.load(f)
            if not data:  # Check if file is empty
                print(f"Warning: File {filepath} is empty.")
                return None
            df = pd.DataFrame(data)
            if 'time' not in df.columns:
                print(f"Warning: 'time' column missing in {filepath}. Skipping.")
                return None
            # Ensure time is numeric (if not already) and sort
            df['time'] = pd.to_numeric(df['time'])
            df = df.sort_values(by='time').reset_index(drop=True)
            # No NaN filling needed if only using OHLCV, but keep it for robustness
            df = df.fillna(method='bfill').fillna(method='ffill')  # Backfill then forward fill
            df = df.dropna()  # Drop any remaining NaNs if ffill/bfill didn't cover edges
            return df
    except json.JSONDecodeError:
        print(f"Error: Could not decode JSON from {filepath}. Skipping.")
        return None
    except Exception as e:
        print(f"Error loading {filepath}: {e}")
        return None

def preprocess_data(df, feature_columns, target_column, window_size, forecast_size, test_size=0.1):
    """Scales data and creates sequences for a single symbol with train/test split."""
    if df is None or df.empty:
        return None, None, None, None, None, None

    MIN_DATA_POINTS = window_size + forecast_size + 10
    if len(df) < MIN_DATA_POINTS:
        print(f"Warning: Insufficient data points ({len(df)} < {MIN_DATA_POINTS}). Skipping preprocessing.")
        return None, None, None, None, None, None

    # Ensure correct column order
    try:
        df_features = df[feature_columns].copy()
    except KeyError as e:
        print(f"Error: Missing expected feature column: {e}. Skipping preprocessing.")
        return None, None, None, None, None, None

    target_col_index = feature_columns.index(target_column)

    # Scale the features
    scaler = MinMaxScaler(feature_range=(0, 1))
    scaled_data = scaler.fit_transform(df_features)

    # Split into training and testing data
    test_samples = int(len(scaled_data) * test_size)
    X_train, y_train, X_test, y_test = [], [], [], []

    # Iterate up to the point where a forecast is possible, split train/test data.
    for i in range(window_size, len(scaled_data) - forecast_size + 1):
        if i < len(scaled_data) - test_samples - forecast_size + 1:
            X_train.append(scaled_data[i - window_size:i, :])
            y_train.append(scaled_data[i + forecast_size - 1, target_col_index])
        else:
            X_test.append(scaled_data[i - window_size:i, :])
            y_test.append(scaled_data[i + forecast_size - 1, target_col_index])

    # Check for empty sequences
    if not X_train or not y_train:
        print("Warning: No training sequences created after preprocessing (likely due to data length).")
        X_train, y_train = None, None  # Ensure these are None for skipping
    if not X_test or not y_test:
        print("Warning: No testing sequences created after preprocessing (likely due to data length).")
        X_test, y_test = None, None

    # Convert to numpy arrays only if data exists.
    if X_train is not None:
        X_train = np.array(X_train)
    if y_train is not None:
        y_train = np.array(y_train)
    if X_test is not None:
        X_test = np.array(X_test)
    if y_test is not None:
        y_test = np.array(y_test)

    return X_train, y_train, X_test, y_test, scaler, target_col_index

def build_model(input_shape):
    """Builds and compiles the LSTM model."""
    model = Sequential()
    model.add(LSTM(64, return_sequences=True, input_shape=input_shape))  #  Units
    model.add(Dropout(0.3))  # Increased dropout
    model.add(LSTM(32, return_sequences=False))  # Second LSTM layer
    model.add(Dropout(0.3))  # Increased dropout
    model.add(Dense(16, activation='relu'))  #  Units
    model.add(Dense(1))  # Output layer for predicting one value
    model.compile(optimizer=Adam(learning_rate=0.0005), loss='mean_squared_error') #Reduced rate
    return model

# --- Main Training Function ---
def train_on_symbol(filepath, window_size, forecast_size):
    """Trains the model on a single symbol's data."""
    symbol = clean_symbol_name(os.path.basename(filepath))  # Extract symbol from filename
    print(f"\n--- Processing Symbol: {symbol} with WINDOW_SIZE = {window_size}, FORECAST_SIZE = {forecast_size} ---")

    # Define model and scaler paths for this symbol
    model_file = os.path.join(MODEL_SAVE_DIR, f'model_{symbol}_w{window_size}_f{forecast_size}.h5')
    scaler_file = os.path.join(MODEL_SAVE_DIR, f'scaler_{symbol}_w{window_size}_f{forecast_size}.pkl')
    target_index_file = os.path.join(MODEL_SAVE_DIR, f'target_index_{symbol}_w{window_size}_f{forecast_size}.json')

    # Load data for the current symbol
    df = load_data_for_symbol(filepath)
    if df is None:
        return False

    # Preprocess data
    print(f"Preprocessing data for {symbol}...")
    X_train, y_train, X_test, y_test, scaler, target_col_index = preprocess_data(
        df, FEATURES, TARGET_COLUMN, window_size, forecast_size, TEST_SIZE
    )

    if X_train is None or y_train is None or scaler is None:
        print(f"Skipping training for {symbol} due to preprocessing issues.")
        return False

    print(f"Data shapes - X_train: {X_train.shape}, y_train: {y_train.shape}")
    if X_test is not None:
        print(f"Data shapes - X_test: {X_test.shape}, y_test: {y_test.shape}")

    # Build or load model
    if os.path.exists(model_file):
        print(f"Loading existing model: {model_file}")
        try:
            model = load_model(model_file)
        except Exception as e:
            print(f"Error loading model {model_file}: {e}. Building a new one.")
            model = build_model((X_train.shape[1], X_train.shape[2]))
    else:
        print("Building new model...")
        model = build_model((X_train.shape[1], X_train.shape[2]))

    model.summary()  # Print model summary

    # Define EarlyStopping callback
    early_stopping = EarlyStopping(
        monitor='val_loss',  # Monitor validation loss
        patience=10,          # Number of epochs to wait before stopping
        restore_best_weights=True # Restore model weights from the epoch with the lowest validation loss
    )

    # Train the model
    print(f"Training model for {symbol}...")
    history = model.fit(
        X_train, y_train,
        epochs=EPOCHS,
        batch_size=BATCH_SIZE,
        validation_split=VALIDATION_SPLIT,
        verbose=1,
        shuffle=True,  # Shuffle data for better training
        callbacks=[early_stopping]  # Add EarlyStopping callback
    )

    # Evaluate the model if test data is available
    if X_test is not None and y_test is not None:
        print("Evaluating model on test data...")
        loss = model.evaluate(X_test, y_test, verbose=0)
        print(f"Test loss: {loss}")

    # Save the model, scaler, and target index
    print(f"Saving model, scaler, and target index for {symbol}...")
    model.save(model_file)
    joblib.dump(scaler, scaler_file)
    with open(target_index_file, 'w') as f:
        json.dump({'target_column_index': target_col_index, 'features': FEATURES}, f)

    print(f"Successfully processed and saved artifacts for {symbol}.")
    return True

# --- Main Execution ---
if __name__ == "__main__":
    # Set up argument parser
    parser = argparse.ArgumentParser(description='Train LSTM model on financial data.')
    parser.add_argument('filepath', type=str, help='Path to the JSON file containing data.')
    parser.add_argument('--window_size', type=int, default=30, help='Window size for LSTM (default: 30)')  # Reduced default
    parser.add_argument('--forecast_size', type=int, default=1, help='Forecast size for LSTM (default: 1)')   # Reduced default
    args = parser.parse_args()

    print("Starting training process...")
    success = train_on_symbol(args.filepath, args.window_size, args.forecast_size)

    if success:
        print("\n--- Training complete. ---")
    else:
        print("\n--- Training failed. Check logs for errors. ---")

# Example:
# python3 train_single.py data/BTCUSDT1d.json --window_size=1 --forecast_size 1
# python3 train_single.py data/BTCUSDT1d.json --window_size=3 --forecast_size 1
# python3 train_single.py data/BTCUSDT1d.json --window_size=5 --forecast_size 1
# python3 train_single.py data/BTCUSDT1d.json --window_size=10 --forecast_size 1
