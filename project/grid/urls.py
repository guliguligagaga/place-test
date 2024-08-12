from django.urls import path

from grid import views

urlpatterns = [
    path("", views.get_canvas_data, name="data"),
]
