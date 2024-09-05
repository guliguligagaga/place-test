from django.shortcuts import render, HttpResponseRedirect
from .forms import SignUpForm, LoginForm
from django.contrib.auth import authenticate, login, logout
from django.contrib.auth.decorators import login_required
from django.contrib import messages

# Create your views here.

# Home page
@login_required
def home(request):
    return render(request, "home.html")


# Dashboard
def dashboard(request):
    if request.user.is_authenticated:
        user = request.user
        full_name = user.get_full_name()
        return render(
            request,
            "dashboard.html",
            {"full_name": full_name},
        )
    else:
        return HttpResponseRedirect("/login/")


# Sigup
def user_signup(request):
    if not request.user.is_authenticated:
        if request.method == "POST":
            form = SignUpForm(request.POST)
            if form.is_valid():
                form.save()
                messages.success(
                    request, "Congratulations!! You have become an Author."
                )
                HttpResponseRedirect("/login/")
        else:
            form = SignUpForm()
        return render(request, "signup.html", {"form": form})
    else:
        return HttpResponseRedirect("/dashboard/")


# Login
def user_login(request):
    if not request.user.is_authenticated:
        if request.method == "POST":
            form = LoginForm(request=request, data=request.POST)
            if form.is_valid():
                uname = form.cleaned_data["username"]
                upass = form.cleaned_data["password"]
                user = authenticate(username=uname, password=upass)
                if user is not None:
                    login(request, user)
                    messages.success(request, "Logged in Successfully !!")
                    return HttpResponseRedirect("/dashboard/")
        else:
            form = LoginForm()
        return render(request, "login.html", {"form": form})
    else:
        return HttpResponseRedirect("/dashboard/")


# Logout
def user_logout(request):
    logout(request)
    return HttpResponseRedirect("/login/")
