(function($) {
    $("#editbtn").click(function(e) {
        e.preventDefault();
        e.stopPropagation();

        $("#view").hide();
        $("#edit").show();
    });
    $("#editcancel").click(function(e) {
        e.preventDefault();
        e.stopPropagation();

        $("#view").show();
        $("#edit").hide();
    });

    $("#createbtn").click(function(e) {
        e.preventDefault();
        e.stopPropagation();

        $("#createbtn").hide();
        $("#create").show();
    });
    $("#create").submit(function(e) {
        e.preventDefault();
        e.stopPropagation();

        var form = $("#create");
        var action = form.attr("action");
        $.ajax(action, {
            "data": form.serialize(),
            "type": "PUT",
            "success": function() {
                // TODO(light): redirect user to project page
            },
        });
    });
    $("#createcancel").click(function(e) {
        e.preventDefault();
        e.stopPropagation();

        // TODO: reset form

        $("#createbtn").show();
        $("#create").hide();
    });
})($);
