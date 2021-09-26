import requests
import argparse
from base64 import b64encode
import yaml

parser = argparse.ArgumentParser()
parser.add_argument('-n', '--n-delay', help='Number of delay seconds before performing an action', required=True)
parser.add_argument('-i', '--iterations', help='Number of iterations to perform', required=True)
args = parser.parse_args()



def load_config_file():
	with open('./config.yaml') as file:
		return yaml.safe_load(file)



def get_issues(token):

    HEADERS = {
         "Accept": "application/vnd.github.v3+json"
    }

    response = requests.get('https://api.github.com/repos/jasperan/github-utils/issues', headers=HEADERS, auth=('jasperan', token))
    print(response.status_code)
    assert response.status_code == 200
    print('Obtained {} issues'.format(len(response.json())))

    for x in response.json():
        print(x.get('url'))



def post_issue(token, iteration):

    HEADERS = {
         "Accept": "application/vnd.github.v3+json"
    }
    DATA = '{}{}{}'.format('{"title":"', iteration, '"}')


    response = requests.post('https://api.github.com/repos/jasperan/github-utils/issues', headers=HEADERS, data=DATA, auth=('jasperan', token))
    print(response.status_code)
    assert response.status_code == 201

    print('Posted {}'.format(response.json().get('url')))
    return response.json().get('number')



def close_issue(token, issue_number):
    HEADERS = {
         "Accept": "application/vnd.github.v3+json"
    }
    DATA = '{"state":"closed"}'


    response = requests.patch('https://api.github.com/repos/jasperan/github-utils/issues/{}'.format(issue_number), headers=HEADERS, data=DATA, auth=('jasperan', token))
    print(response.status_code)
    assert response.status_code == 200
    print('Closed #{}'.format(issue_number))
    return response.status_code


def main():
    data = load_config_file()
    personal_access_token = data['personal_access_token']
    #get_issues(personal_access_token)
    print('Number of iterations: {}'.format(args.iterations))
    for i in range(int(args.iterations)):
        num = post_issue(personal_access_token, i)
        close_issue(personal_access_token, num)



if __name__ == '__main__':
    main()