import json
import requests

class json_handler:
    
    def __init__(self,url):
        self.url = url
        self.json_data = None
        self.refresh_data()
        return
    
    
    def refresh_data(self):
        response = requests.get(self.url)
        self.json_data = json.loads(response.text);
        
        return True
    
    def get_number_of_players(self):
        return self.json_data['num_of_players']
    
    def get_name(self,player_num):
        return self.json_data["player"][player_num]["name"]
        
    def get_hand(self,player_num):
        return self.json_data['player'][player_num]['hand']
    
    def get_purse(self,player_num):
        return self.json_data['player'][player_num]['purse']
    
    def get_bet(self,player_num):
        return self.json_data['player'][player_num]['bet']
    
    def get_playing(self, player_num):
        player = self.json_data['active_player']
        player -= 1
        return player
    
    def get_fold(self, player_num):
        return self.json_data['player'][player_num]['fold']
    
    def get_round(self):
        return self.json_data['round']
    
    def get_valid_buttons(self):
        valid_moves = None
        
        moves = {}
        i = 0
        no_error = True
        while no_error:
            try:
                move = self.json_data['validMoves'][i]['move']
                name = self.json_data['validMoves'][i]['name']
                moves[move] = name
                i += 1
            except:
                no_error=False     
        
        return moves
    
    