import json
import os
import random
import time
import logging

logging.basicConfig(level=logging.DEBUG, format='%(asctime)s - %(levelname)s - %(message)s')
logging.getLogger()


path = 'config.json'
logging.info(os.path.abspath(path))
with open(path, 'r') as f:
    config = json.load(f)

for i in range(10):
    time.sleep(1)
    logging.info(config['Task1']['Group3']['setting1'])
    logging.info(config['Task1']['Group3']['setting2'])
    logging.info(config['Task1']['Group3']['setting3'])
    if random.random() < 0.05:
        logging.error(f'Error occurred')
        raise Exception("Error occurred")
