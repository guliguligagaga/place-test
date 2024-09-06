import json

import requests
from django.http import HttpResponse
from django.shortcuts import HttpResponseRedirect



def get_canvas_data(request):
    # Dummy data for illustration
    data = requests.get('http://rust_backend:8080/api/grid')

    # Convert hex string to bytes
    binary_data = data.content
    return HttpResponse(binary_data, content_type='application/octet-stream')


def draw(request):
    # Dummy data for illustration
    data = requests.post('http://rust_backend:8080/api/draw',json=json.loads(request.POST['myData']))

    return HttpResponseRedirect("/dashboard/")
