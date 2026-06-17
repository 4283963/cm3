CREATE DATABASE IF NOT EXISTS supercharger
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE supercharger;

CREATE TABLE IF NOT EXISTS chargers (
  id INT PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  max_power DOUBLE NOT NULL DEFAULT 120,
  current_power DOUBLE NOT NULL DEFAULT 0,
  status VARCHAR(20) NOT NULL DEFAULT 'idle',
  last_update DATETIME DEFAULT CURRENT_TIMESTAMP,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vehicles (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  charger_id INT NOT NULL,
  vin VARCHAR(50),
  license_plate VARCHAR(20),
  battery_capacity DOUBLE,
  current_soc DOUBLE,
  max_accept_power DOUBLE,
  allocated_power DOUBLE DEFAULT 0,
  target_soc DOUBLE DEFAULT 100,
  start_time DATETIME,
  estimated_end_time DATETIME,
  status VARCHAR(20) NOT NULL DEFAULT 'idle',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_charger_id (charger_id),
  INDEX idx_status (status),
  INDEX idx_vin (vin)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS power_allocation_records (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  timestamp DATETIME NOT NULL,
  charger_id INT NOT NULL,
  vehicle_vin VARCHAR(50),
  current_soc DOUBLE,
  max_power DOUBLE,
  allocated_power DOUBLE,
  total_power DOUBLE,
  reason VARCHAR(200),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_timestamp (timestamp),
  INDEX idx_charger_id (charger_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS station_status (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  timestamp DATETIME,
  total_max_power DOUBLE,
  current_total_power DOUBLE,
  active_chargers INT,
  idle_chargers INT,
  fault_chargers INT,
  total_charging_vehicles INT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_timestamp (timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
