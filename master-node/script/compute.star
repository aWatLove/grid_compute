def compute(input_data):
    """
    Находит маршрут с минимальной стоимостью
    Возвращает: {"status": "ok", "data": {"route": [...], "cost": ...}}
    """
    # Валидация входных данных
    if not isinstance(input_data, dict):
        return {"status": "error", "data": "Input must be a dict"}

    required_fields = ["matrix", "routes"]
    for field in required_fields:
        if field not in input_data:
            return {"status": "error", "data": f"Missing {field}"}

    matrix = input_data["matrix"]
    routes = input_data["routes"]

    # Поиск лучшего маршрута
    best_route = None
    best_cost = None

    for route in routes:
        current_cost = 0
        for i in range(len(route)-1):
            a = route[i]
            b = route[i+1]
            current_cost += matrix[a][b]

        if best_cost is None or current_cost < best_cost:
            best_cost = current_cost
            best_route = route

    return {
        "status": "ok",
        "data": {
            "route": best_route,
            "cost": best_cost
        }
    }