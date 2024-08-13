import json

import requests
from django.http import HttpResponse
from django.utils.crypto import get_random_string


def get_canvas_data(request):
    # Dummy data for illustration
    data = requests.get('http://localhost:8080/api/grid')

    # Convert hex string to bytes
    binary_data = data.content
    return HttpResponse(binary_data, content_type='application/octet-stream')

def draw(request):
    # Dummy data for illustration
    data = requests.post('http://localhost:8080/api/draw',json=json.loads(request.POST['myData']))

    return HttpResponse(data.content)
